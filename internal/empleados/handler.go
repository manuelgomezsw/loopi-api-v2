package empleados

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
)

const otelScope = "loopi-api/empleados"

var tracer = otel.Tracer(otelScope)

// EmpleadoMetrics agrupa los instrumentos OTel del dominio empleados.
type EmpleadoMetrics struct {
	RequestDuration metric.Float64Histogram
	RequestCount    metric.Int64Counter
}

// NewMetrics inicializa los instrumentos OTel del dominio empleados.
func NewMetrics() (*EmpleadoMetrics, error) {
	meter := otel.Meter(otelScope)
	dur, err := meter.Float64Histogram(
		"empleados.request.duration",
		metric.WithDescription("Duración de requests de empleados en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	cnt, err := meter.Int64Counter(
		"empleados.request.count",
		metric.WithDescription("Conteo de requests de empleados por operación y resultado"),
	)
	if err != nil {
		return nil, err
	}
	return &EmpleadoMetrics{RequestDuration: dur, RequestCount: cnt}, nil
}

func noopMetrics() *EmpleadoMetrics {
	m, _ := NewMetrics()
	if m == nil {
		return &EmpleadoMetrics{}
	}
	return m
}

// EmpleadoHandler agrupa los handlers HTTP del módulo de empleados.
type EmpleadoHandler struct {
	svc     EmpleadoService
	metrics *EmpleadoMetrics
}

// NewHandler crea un EmpleadoHandler con métricas no-op.
func NewHandler(svc EmpleadoService) *EmpleadoHandler {
	return &EmpleadoHandler{svc: svc, metrics: noopMetrics()}
}

// NewHandlerWithMetrics crea el handler con métricas OTel configuradas.
func NewHandlerWithMetrics(svc EmpleadoService, m *EmpleadoMetrics) *EmpleadoHandler {
	return &EmpleadoHandler{svc: svc, metrics: m}
}

// RegisterRoutes registra los endpoints de empleados. El endpoint de cambio de contraseña
// solo requiere JWT válido (no solo_admin), todos los demás requieren rol=admin.
func (h *EmpleadoHandler) RegisterRoutes(mux *http.ServeMux, jwtMiddleware func(http.Handler) http.Handler) {
	admin := func(hf http.HandlerFunc) http.Handler {
		return jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if claims == nil {
				writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
				return
			}
			if claims.Rol != "admin" {
				logRequest(userIDFromClaims(claims), claims.Rol, "acceso_denegado", r.URL.Path, http.StatusForbidden, 0)
				writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "No tienes permiso para esta operación."})
				return
			}
			hf(w, r)
		}))
	}
	jwt := func(hf http.HandlerFunc) http.Handler { return jwtMiddleware(hf) }

	mux.Handle("POST /api/v1/empleados", admin(h.Crear))
	mux.Handle("GET /api/v1/empleados", admin(h.Listar))
	mux.Handle("GET /api/v1/empleados/{id}", admin(h.ObtenerPorID))
	mux.Handle("PUT /api/v1/empleados/{id}", admin(h.Actualizar))
	mux.Handle("PATCH /api/v1/empleados/{id}/estado", admin(h.CambiarEstado))
	mux.Handle("POST /api/v1/empleados/{id}/contrasena", admin(h.ResetearContrasena))
	// Solo JWT válido — accesible con requiere_cambio_contrasena=1 (RF-EMP-04.6).
	mux.Handle("POST /api/v1/empleados/{id}/contrasena/cambiar", jwt(h.CambiarContrasena))
}

// Crear maneja POST /api/v1/empleados.
func (h *EmpleadoHandler) Crear(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	var req CrearEmpleadoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	resp, err := h.svc.CrearEmpleado(ctx, actorID, req)
	if err != nil {
		status, body := mapServiceError(err)
		logRequest(actorID, claims.Rol, "crear_empleado", "", status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, claims.Rol, "crear_empleado", fmt.Sprintf("%d", resp.ID), http.StatusCreated, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusCreated), attribute.Int64("empleado.id", int64(resp.ID)))
	h.recordMetrics(ctx, "crear_empleado", http.StatusCreated, durMs)
	writeJSON(w, http.StatusCreated, resp)
}

// Listar maneja GET /api/v1/empleados.
func (h *EmpleadoHandler) Listar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	p := ListarEmpleadosParams{
		Q:     r.URL.Query().Get("q"),
		Page:  parseIntQuery(r, "page", 1),
		Limit: parseIntQuery(r, "limit", 20),
	}
	if tid := r.URL.Query().Get("tienda_id"); tid != "" {
		if v, err := strconv.ParseUint(tid, 10, 64); err == nil {
			p.TiendaID = &v
		}
	}
	if estado := r.URL.Query().Get("estado"); estado != "" {
		estadosValidos := map[string]bool{"activo": true, "inactivo": true, "todos": true}
		if !estadosValidos[estado] {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "estado_invalido", Mensaje: "El estado debe ser 'activo', 'inactivo' o 'todos'.", Campo: "estado"})
			return
		}
		if estado != "todos" {
			v := estado == "activo"
			p.Activo = &v
		}
	}

	resp, err := h.svc.ListarEmpleados(ctx, p)
	if err != nil {
		logRequest(actorID, claims.Rol, "listar_empleados", "", http.StatusInternalServerError, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", http.StatusInternalServerError))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."})
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, claims.Rol, "listar_empleados", "", http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	h.recordMetrics(ctx, "listar_empleados", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerPorID maneja GET /api/v1/empleados/{id}.
func (h *EmpleadoHandler) ObtenerPorID(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.obtener",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados/{id}")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, err := h.svc.ObtenerEmpleado(ctx, id)
	if err != nil {
		status, body := mapServiceError(err)
		logRequest(actorID, claims.Rol, "obtener_empleado", fmt.Sprintf("%d", id), status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, claims.Rol, "obtener_empleado", fmt.Sprintf("%d", id), http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("empleado.id", int64(id)))
	h.recordMetrics(ctx, "obtener_empleado", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Actualizar maneja PUT /api/v1/empleados/{id}.
func (h *EmpleadoHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.actualizar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados/{id}")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req EditarEmpleadoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	resp, err := h.svc.EditarEmpleado(ctx, actorID, id, req)
	if err != nil {
		status, body := mapServiceError(err)
		logRequest(actorID, claims.Rol, "editar_empleado", fmt.Sprintf("%d", id), status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, claims.Rol, "editar_empleado", fmt.Sprintf("%d", id), http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("empleado.id", int64(id)))
	h.recordMetrics(ctx, "editar_empleado", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// CambiarEstado maneja PATCH /api/v1/empleados/{id}/estado.
func (h *EmpleadoHandler) CambiarEstado(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.cambiar_estado",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados/{id}/estado")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req CambiarEstadoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	resp, err := h.svc.CambiarEstado(ctx, actorID, id, req.Activo)
	if err != nil {
		status, body := mapServiceError(err)
		logRequest(actorID, claims.Rol, "cambiar_estado_empleado", fmt.Sprintf("%d", id), status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, claims.Rol, "cambiar_estado_empleado", fmt.Sprintf("%d", id), http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("empleado.id", int64(id)))
	h.recordMetrics(ctx, "cambiar_estado_empleado", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, struct {
		ID     uint64 `json:"id"`
		Activo bool   `json:"activo"`
	}{ID: resp.ID, Activo: resp.Activo})
}

// ResetearContrasena maneja POST /api/v1/empleados/{id}/contrasena.
func (h *EmpleadoHandler) ResetearContrasena(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.reset_contrasena",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados/{id}/contrasena")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	contrasenaTemp, err := h.svc.ResetearContrasena(ctx, actorID, id)
	if err != nil {
		status, body := mapServiceError(err)
		logRequest(actorID, claims.Rol, "reset_contrasena", fmt.Sprintf("%d", id), status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, claims.Rol, "reset_contrasena", fmt.Sprintf("%d", id), http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("empleado.id", int64(id)))
	h.recordMetrics(ctx, "reset_contrasena", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, ResetContrasenaResponse{ContrasenaTemporal: contrasenaTemp})
}

// CambiarContrasena maneja POST /api/v1/empleados/{id}/contrasena/cambiar.
// Solo requiere JWT válido — no requiere rol=admin (RF-EMP-04.6).
func (h *EmpleadoHandler) CambiarContrasena(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "empleados.cambiar_contrasena",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/empleados/{id}/contrasena/cambiar")),
	)
	defer span.End()
	inicio := time.Now()
	claims := auth.ClaimsFromContext(ctx)
	actorID := userIDFromClaims(claims)

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req CambiarContrasenaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	if err := h.svc.CambiarContrasena(ctx, id, req.NuevaContrasena); err != nil {
		status, body := mapServiceError(err)
		logRequest(actorID, "", "cambiar_contrasena", fmt.Sprintf("%d", id), status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(actorID, "", "cambiar_contrasena", fmt.Sprintf("%d", id), http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	h.recordMetrics(ctx, "cambiar_contrasena", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, struct{ Mensaje string `json:"mensaje"` }{"Contraseña actualizada correctamente."})
}

// --- Utilidades ---

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func parseIDPath(r *http.Request) (uint64, error) {
	return strconv.ParseUint(r.PathValue("id"), 10, 64)
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

func userIDFromClaims(claims *auth.Claims) uint64 {
	if claims == nil {
		return 0
	}
	id, _ := strconv.ParseUint(claims.Subject, 10, 64)
	return id
}

// logRequest emite un log estructurado JSON a stdout (RF-EMP-05-A.4 + Constitución VI).
func logRequest(userID uint64, rol, operacion, empleadoID string, statusHTTP int, duracion time.Duration) {
	log.Printf(
		`{"user_id":%d,"rol":"%s","operacion":"%s","empleado_id":"%s","status_http":%d,"duracion_ms":%d}`,
		userID, rol, operacion, empleadoID, statusHTTP, duracion.Milliseconds(),
	)
}

func mapServiceError(err error) (int, errorResponse) {
	var valErr *ValidationError
	switch {
	case errors.As(err, &valErr):
		return http.StatusUnprocessableEntity, errorResponse{Error: valErr.Codigo, Mensaje: valErr.Mensaje, Campo: valErr.Campo}
	case errors.Is(err, ErrEmpleadoNoEncontrado):
		return http.StatusNotFound, errorResponse{Error: "empleado_no_encontrado", Mensaje: "El empleado no existe."}
	case errors.Is(err, ErrUsuarioDuplicado):
		return http.StatusConflict, errorResponse{Error: "usuario_duplicado", Mensaje: "Ya existe un empleado con ese nombre de usuario.", Campo: "usuario"}
	case errors.Is(err, ErrUltimoAdminActivo):
		return http.StatusUnprocessableEntity, errorResponse{Error: "ultimo_admin_activo", Mensaje: "No es posible inactivar al último administrador activo."}
	case errors.Is(err, ErrTiendaNoExiste):
		return http.StatusUnprocessableEntity, errorResponse{Error: "tienda_no_existe", Mensaje: "La tienda no existe o está inactiva.", Campo: "tienda_id"}
	case errors.Is(err, ErrTiendaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "tienda_no_existe", Mensaje: "La tienda no existe o está inactiva.", Campo: "tienda_id"}
	default:
		return http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."}
	}
}

func (h *EmpleadoHandler) recordMetrics(ctx context.Context, operacion string, status int, durMs float64) {
	if h.metrics == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("operacion", operacion),
		attribute.Int("status_http", status),
	)
	if h.metrics.RequestDuration != nil {
		h.metrics.RequestDuration.Record(ctx, durMs, attrs)
	}
	if h.metrics.RequestCount != nil {
		h.metrics.RequestCount.Add(ctx, 1, attrs)
	}
}
