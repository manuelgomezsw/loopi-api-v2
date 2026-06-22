package tiendas

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

const otelScope = "loopi-api/tiendas"

var tracer = otel.Tracer(otelScope)

// TiendaMetrics agrupa los instrumentos OTel del dominio tiendas.
type TiendaMetrics struct {
	RequestDuration metric.Float64Histogram
	RequestCount    metric.Int64Counter
}

// NewMetrics inicializa los instrumentos OTel del dominio tiendas.
func NewMetrics() (*TiendaMetrics, error) {
	meter := otel.Meter(otelScope)

	dur, err := meter.Float64Histogram(
		"tiendas.request.duration",
		metric.WithDescription("Duración de requests de tiendas en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	cnt, err := meter.Int64Counter(
		"tiendas.request.count",
		metric.WithDescription("Conteo de requests de tiendas por operación y resultado"),
	)
	if err != nil {
		return nil, err
	}

	return &TiendaMetrics{RequestDuration: dur, RequestCount: cnt}, nil
}

// noopMetrics retorna métricas no-op (se usan cuando OTel no está configurado).
func noopMetrics() *TiendaMetrics {
	m, _ := NewMetrics()
	if m == nil {
		return &TiendaMetrics{}
	}
	return m
}

// TiendaHandler agrupa los handlers HTTP del módulo de tiendas.
type TiendaHandler struct {
	svc     TiendaService
	metrics *TiendaMetrics
}

// NewTiendaHandler crea un nuevo TiendaHandler con métricas no-op.
func NewTiendaHandler(svc TiendaService) *TiendaHandler {
	return &TiendaHandler{svc: svc, metrics: noopMetrics()}
}

// NewTiendaHandlerWithMetrics crea el handler con métricas OTel configuradas.
func NewTiendaHandlerWithMetrics(svc TiendaService, m *TiendaMetrics) *TiendaHandler {
	return &TiendaHandler{svc: svc, metrics: m}
}

// RegisterRoutes registra los 6 endpoints de tiendas sobre el ServeMux de Go 1.22+.
func (h *TiendaHandler) RegisterRoutes(mux *http.ServeMux, jwtMiddleware func(http.Handler) http.Handler) {
	wrap := func(handler http.HandlerFunc) http.Handler {
		return jwtMiddleware(handler)
	}
	mux.Handle("POST /api/v1/tiendas", wrap(h.Crear))
	mux.Handle("GET /api/v1/tiendas", wrap(h.Listar))
	mux.Handle("GET /api/v1/tiendas/{id}", wrap(h.ObtenerPorID))
	mux.Handle("PUT /api/v1/tiendas/{id}", wrap(h.Actualizar))
	mux.Handle("POST /api/v1/tiendas/{id}/inactivar", wrap(h.Inactivar))
	mux.Handle("POST /api/v1/tiendas/{id}/reactivar", wrap(h.Reactivar))
}

// --- Handlers ---

// Listar maneja GET /api/v1/tiendas.
func (h *TiendaHandler) Listar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "tiendas.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/tiendas")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	adminID, ok := adminIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		logRequest(adminID, claims.Rol, "listar_tiendas", 0, http.StatusForbidden, time.Since(inicio))
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "No tienes permiso para esta operación."})
		return
	}

	estado := r.URL.Query().Get("estado")
	if estado == "" {
		estado = "todos"
	}
	pagina := parseIntQuery(r, "pagina", 1)
	limite := parseIntQuery(r, "limite", 50)

	resp, err := h.svc.Listar(estado, pagina, limite)
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			logRequest(adminID, claims.Rol, "listar_tiendas", 0, http.StatusBadRequest, time.Since(inicio))
			span.SetAttributes(attribute.Int("http.status_code", http.StatusBadRequest))
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: valErr.Codigo, Mensaje: valErr.Mensaje, Campo: valErr.Campo})
			return
		}
		logRequest(adminID, claims.Rol, "listar_tiendas", 0, http.StatusInternalServerError, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", http.StatusInternalServerError))
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."})
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(adminID, claims.Rol, "listar_tiendas", 0, http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK))
	h.recordMetrics(ctx, "listar_tiendas", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Crear maneja POST /api/v1/tiendas.
func (h *TiendaHandler) Crear(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "tiendas.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/tiendas")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	adminID, ok := adminIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		logRequest(adminID, claims.Rol, "crear_tienda", 0, http.StatusForbidden, time.Since(inicio))
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "No tienes permiso para esta operación."})
		return
	}

	var req TiendaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}
	if campo, falta := firstEmptyField([]fieldCheck{
		{"codigo", req.Codigo}, {"nombre", req.Nombre},
		{"direccion", req.Direccion}, {"ciudad", req.Ciudad}, {"telefono", req.Telefono},
	}); falta {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El campo es obligatorio.", Campo: campo})
		return
	}

	resp, err := h.svc.Crear(req, adminID)
	if err != nil {
		status, body := mapServiceError(err)
		logRequest(adminID, claims.Rol, "crear_tienda", 0, status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(adminID, claims.Rol, "crear_tienda", resp.ID, http.StatusCreated, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusCreated), attribute.Int64("tienda.id", int64(resp.ID)))
	h.recordMetrics(ctx, "crear_tienda", http.StatusCreated, durMs)
	writeJSON(w, http.StatusCreated, resp)
}

// ObtenerPorID maneja GET /api/v1/tiendas/{id}.
func (h *TiendaHandler) ObtenerPorID(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "tiendas.obtener",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/tiendas/{id}")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	adminID, ok := adminIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		logRequest(adminID, claims.Rol, "obtener_tienda", 0, http.StatusForbidden, time.Since(inicio))
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "No tienes permiso para esta operación."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.ObtenerPorID(id)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		logRequest(adminID, claims.Rol, "obtener_tienda", id, status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(adminID, claims.Rol, "obtener_tienda", id, http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("tienda.id", int64(id)))
	h.recordMetrics(ctx, "obtener_tienda", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Actualizar maneja PUT /api/v1/tiendas/{id}.
func (h *TiendaHandler) Actualizar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "tiendas.actualizar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/tiendas/{id}")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	adminID, ok := adminIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		logRequest(adminID, claims.Rol, "actualizar_tienda", 0, http.StatusForbidden, time.Since(inicio))
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "No tienes permiso para esta operación."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req TiendaUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}
	if campo, falta := firstEmptyField([]fieldCheck{
		{"nombre", req.Nombre}, {"direccion", req.Direccion},
		{"ciudad", req.Ciudad}, {"telefono", req.Telefono},
	}); falta {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "El campo es obligatorio.", Campo: campo})
		return
	}

	resp, svcErr := h.svc.Actualizar(id, req, adminID)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		logRequest(adminID, claims.Rol, "actualizar_tienda", id, status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(adminID, claims.Rol, "actualizar_tienda", id, http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("tienda.id", int64(id)))
	h.recordMetrics(ctx, "actualizar_tienda", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Inactivar maneja POST /api/v1/tiendas/{id}/inactivar.
func (h *TiendaHandler) Inactivar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "tiendas.inactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/tiendas/{id}/inactivar")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	adminID, ok := adminIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		logRequest(adminID, claims.Rol, "inactivar_tienda", 0, http.StatusForbidden, time.Since(inicio))
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "No tienes permiso para esta operación."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.Inactivar(id, adminID)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		logRequest(adminID, claims.Rol, "inactivar_tienda", id, status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(adminID, claims.Rol, "inactivar_tienda", id, http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("tienda.id", int64(id)))
	h.recordMetrics(ctx, "inactivar_tienda", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Reactivar maneja POST /api/v1/tiendas/{id}/reactivar.
func (h *TiendaHandler) Reactivar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "tiendas.reactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/tiendas/{id}/reactivar")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	adminID, ok := adminIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		logRequest(adminID, claims.Rol, "reactivar_tienda", 0, http.StatusForbidden, time.Since(inicio))
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "No tienes permiso para esta operación."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.Reactivar(id, adminID)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		logRequest(adminID, claims.Rol, "reactivar_tienda", id, status, time.Since(inicio))
		span.SetAttributes(attribute.Int("http.status_code", status))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logRequest(adminID, claims.Rol, "reactivar_tienda", id, http.StatusOK, time.Since(inicio))
	span.SetAttributes(attribute.Int("http.status_code", http.StatusOK), attribute.Int64("tienda.id", int64(id)))
	h.recordMetrics(ctx, "reactivar_tienda", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// --- Utilidades ---

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
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

// adminIDFromClaims extrae el user_id del Subject del JWT como uint64.
func adminIDFromClaims(claims *auth.Claims) (uint64, bool) {
	if claims == nil {
		return 0, false
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

type fieldCheck struct {
	nombre string
	valor  string
}

// firstEmptyField retorna el nombre del primer campo vacío encontrado.
func firstEmptyField(fields []fieldCheck) (string, bool) {
	for _, f := range fields {
		if f.valor == "" {
			return f.nombre, true
		}
	}
	return "", false
}

// mapServiceError convierte errores del service en códigos HTTP y bodies de respuesta.
func mapServiceError(err error) (int, errorResponse) {
	switch {
	case errors.Is(err, ErrTiendaNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "tienda_no_encontrada", Mensaje: "La tienda no existe."}
	case errors.Is(err, ErrNombreDuplicado):
		return http.StatusConflict, errorResponse{Error: "nombre_duplicado", Mensaje: "Ya existe una tienda con ese nombre.", Campo: "nombre"}
	case errors.Is(err, ErrCodigoDuplicado):
		return http.StatusConflict, errorResponse{Error: "codigo_duplicado", Mensaje: "Ya existe una tienda con ese código.", Campo: "codigo"}
	case errors.Is(err, ErrTiendaYaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "tienda_ya_inactiva", Mensaje: "La tienda ya se encuentra inactiva."}
	case errors.Is(err, ErrTiendaYaActiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "tienda_ya_activa", Mensaje: "La tienda ya se encuentra activa."}
	default:
		return http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."}
	}
}

// logRequest emite un log estructurado JSON a stdout (Constitución Principio VI).
func logRequest(userID uint64, rol, operacion string, tiendaID uint64, statusHTTP int, duracion time.Duration) {
	log.Println(fmt.Sprintf(
		`{"user_id":%d,"rol":"%s","operacion":"%s","tienda_id":%d,"status_http":%d,"duracion_ms":%d}`,
		userID, rol, operacion, tiendaID, statusHTTP, duracion.Milliseconds(),
	))
}

// recordMetrics registra duración y conteo en los histogramas OTel.
func (h *TiendaHandler) recordMetrics(ctx context.Context, operacion string, status int, durMs float64) {
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
