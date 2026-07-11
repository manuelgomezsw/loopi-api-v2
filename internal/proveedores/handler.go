package proveedores

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
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
)

const otelScope = "loopi-api/proveedores"

var tracer = otel.Tracer(otelScope)

var estadosValidos = map[string]bool{"activo": true, "inactivo": true, "todos": true}

// Metrics agrupa los instrumentos OTel del módulo proveedores.
type Metrics struct {
	RequestDuration metric.Float64Histogram
	RequestCount    metric.Int64Counter
}

// NewMetrics inicializa los instrumentos OTel del módulo.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(otelScope)
	dur, err := meter.Float64Histogram(
		"catalogo.proveedor.request.duration",
		metric.WithDescription("Duración de requests de proveedores en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	cnt, err := meter.Int64Counter(
		"catalogo.proveedor.request.total",
		metric.WithDescription("Conteo de requests de proveedores por operación y resultado"),
	)
	if err != nil {
		return nil, err
	}
	return &Metrics{RequestDuration: dur, RequestCount: cnt}, nil
}

func noopMetrics() *Metrics {
	m, _ := NewMetrics()
	if m == nil {
		return &Metrics{}
	}
	return m
}

// Handler agrupa los handlers HTTP del módulo proveedores.
type Handler struct {
	svc     Service
	metrics *Metrics
}

// NewHandler crea un Handler con métricas no-op.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc, metrics: noopMetrics()}
}

// NewHandlerWithMetrics crea el handler con métricas OTel configuradas.
func NewHandlerWithMetrics(svc Service, m *Metrics) *Handler {
	return &Handler{svc: svc, metrics: m}
}

// RegisterRoutes registra todos los endpoints del módulo en el ServeMux.
// Todo el módulo, incluida la lectura, requiere rol admin (HU-1 Escenario 3).
func (h *Handler) RegisterRoutes(mux *http.ServeMux, jwtMiddleware func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return jwtMiddleware(fn) }

	mux.Handle("GET /api/v1/proveedores", wrap(h.Listar))
	mux.Handle("GET /api/v1/proveedores/{id}", wrap(h.ObtenerPorID))
	mux.Handle("POST /api/v1/proveedores", wrap(h.Crear))
	mux.Handle("PUT /api/v1/proveedores/{id}", wrap(h.Editar))
	mux.Handle("PATCH /api/v1/proveedores/{id}/inactivar", wrap(h.Inactivar))
	mux.Handle("PATCH /api/v1/proveedores/{id}/activar", wrap(h.Activar))
}

// --- Handlers ---

// Crear maneja POST /api/v1/proveedores.
func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "proveedores.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/proveedores")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "acceso_denegado")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede gestionar proveedores."})
		return
	}

	var req CrearProveedorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	p, err := h.svc.Crear(&req, userID, claims.Rol)
	if err != nil {
		status, body := mapServiceError(err)
		span.SetStatus(codes.Error, err.Error())
		logOp(userID, claims.Rol, "crear_proveedor", 0, status, time.Since(inicio))
		h.recordMetrics(ctx, "crear_proveedor", status, float64(time.Since(inicio).Milliseconds()))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("proveedor.id", int64(p.ID)))
	logOp(userID, claims.Rol, "crear_proveedor", p.ID, http.StatusCreated, time.Since(inicio))
	h.recordMetrics(ctx, "crear_proveedor", http.StatusCreated, durMs)
	writeJSON(w, http.StatusCreated, p)
}

// Listar maneja GET /api/v1/proveedores.
func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "proveedores.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/proveedores")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "acceso_denegado")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede consultar proveedores."})
		return
	}

	filtros := &FiltrosListado{
		Busqueda: r.URL.Query().Get("busqueda"),
		Page:     parseIntQuery(r, "page", 1),
		Limit:    parseIntQuery(r, "limit", 50),
	}
	if filtros.Limit > 200 {
		filtros.Limit = 200
	}
	if estado := r.URL.Query().Get("estado"); estado != "" {
		if !estadosValidos[estado] {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "estado_invalido", Mensaje: "El estado debe ser 'activo', 'inactivo' o 'todos'.", Campo: "estado"})
			return
		}
		if estado != "todos" {
			b := estado == "activo"
			filtros.Activo = &b
		}
	}

	resp, err := h.svc.Listar(filtros)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."})
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logOp(userID, claims.Rol, "listar_proveedores", 0, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "listar_proveedores", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerPorID maneja GET /api/v1/proveedores/{id}.
func (h *Handler) ObtenerPorID(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "proveedores.obtener",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/proveedores/{id}")),
	)
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	_, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "acceso_denegado")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede consultar proveedores."})
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
		span.SetStatus(codes.Error, svcErr.Error())
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("proveedor.id", int64(id)))
	writeJSON(w, http.StatusOK, resp)
}

// Editar maneja PUT /api/v1/proveedores/{id}.
func (h *Handler) Editar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "proveedores.editar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/proveedores/{id}")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "acceso_denegado")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede editar proveedores."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req EditarProveedorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	p, svcErr := h.svc.Editar(id, &req, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "editar_proveedor", id, status, time.Since(inicio))
		h.recordMetrics(ctx, "editar_proveedor", status, float64(time.Since(inicio).Milliseconds()))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("proveedor.id", int64(id)))
	logOp(userID, claims.Rol, "editar_proveedor", id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "editar_proveedor", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, p)
}

// Inactivar maneja PATCH /api/v1/proveedores/{id}/inactivar.
func (h *Handler) Inactivar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "proveedores.inactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/proveedores/{id}/inactivar")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "acceso_denegado")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede inactivar proveedores."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.Inactivar(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "inactivar_proveedor", id, status, time.Since(inicio))
		h.recordMetrics(ctx, "inactivar_proveedor", status, float64(time.Since(inicio).Milliseconds()))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("proveedor.id", int64(id)))
	logOp(userID, claims.Rol, "inactivar_proveedor", id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "inactivar_proveedor", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Activar maneja PATCH /api/v1/proveedores/{id}/activar.
func (h *Handler) Activar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "proveedores.activar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/proveedores/{id}/activar")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "acceso_denegado")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede activar proveedores."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.Activar(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "activar_proveedor", id, status, time.Since(inicio))
		h.recordMetrics(ctx, "activar_proveedor", status, float64(time.Since(inicio).Milliseconds()))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("proveedor.id", int64(id)))
	logOp(userID, claims.Rol, "activar_proveedor", id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "activar_proveedor", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
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

func userIDFromClaims(claims *auth.Claims) (uint64, bool) {
	if claims == nil {
		return 0, false
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func logOp(userID uint64, rol, operacion string, proveedorID uint64, statusHTTP int, dur time.Duration) {
	log.Println(fmt.Sprintf(
		`{"user_id":%d,"rol":"%s","operacion":"%s","proveedor_id":%d,"status_http":%d,"duracion_ms":%d}`,
		userID, rol, operacion, proveedorID, statusHTTP, dur.Milliseconds(),
	))
}

func (h *Handler) recordMetrics(ctx context.Context, operacion string, status int, durMs float64) {
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

func mapServiceError(err error) (int, errorResponse) {
	var valErr *ValidationError
	switch {
	case errors.As(err, &valErr):
		return http.StatusBadRequest, errorResponse{Error: valErr.Codigo, Mensaje: valErr.Mensaje, Campo: valErr.Campo}
	case errors.Is(err, ErrProveedorNoEncontrado):
		return http.StatusNotFound, errorResponse{Error: "proveedor_no_encontrado", Mensaje: "El proveedor no existe."}
	case errors.Is(err, ErrNITDuplicado):
		return http.StatusConflict, errorResponse{Error: "nit_duplicado", Mensaje: "Ya existe un proveedor con ese NIT.", Campo: "nit"}
	case errors.Is(err, ErrYaInactivo):
		return http.StatusConflict, errorResponse{Error: "ya_inactivo", Mensaje: "El proveedor ya estaba inactivo."}
	case errors.Is(err, ErrYaActivo):
		return http.StatusConflict, errorResponse{Error: "ya_activo", Mensaje: "El proveedor ya estaba activo."}
	default:
		return http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."}
	}
}
