package unidades_medida

import (
	"context"
	"encoding/json"
	"errors"
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

const otelScope = "loopi-api/unidades_medida"

var tracer = otel.Tracer(otelScope)

// UMMetrics agrupa los instrumentos OTel del módulo unidades_medida.
type UMMetrics struct {
	RequestDuration metric.Float64Histogram
	RequestCount    metric.Int64Counter
}

// NewMetrics inicializa los instrumentos OTel del módulo.
func NewMetrics() (*UMMetrics, error) {
	meter := otel.Meter(otelScope)
	dur, err := meter.Float64Histogram(
		"unidades_medida.request.duration",
		metric.WithDescription("Duración de requests de unidades_medida en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	cnt, err := meter.Int64Counter(
		"unidades_medida.request.total",
		metric.WithDescription("Conteo de requests de unidades_medida por operación y resultado"),
	)
	if err != nil {
		return nil, err
	}
	return &UMMetrics{RequestDuration: dur, RequestCount: cnt}, nil
}

func noopMetrics() *UMMetrics {
	m, _ := NewMetrics()
	if m == nil {
		return &UMMetrics{}
	}
	return m
}

// UMHandler agrupa los handlers HTTP del módulo unidades_medida.
type UMHandler struct {
	svc     UMService
	metrics *UMMetrics
}

// NewHandler crea un UMHandler con métricas no-op.
func NewHandler(svc UMService) *UMHandler {
	return &UMHandler{svc: svc, metrics: noopMetrics()}
}

// NewHandlerWithMetrics crea el handler con métricas OTel configuradas.
func NewHandlerWithMetrics(svc UMService, m *UMMetrics) *UMHandler {
	return &UMHandler{svc: svc, metrics: m}
}

// RegisterRoutes registra todos los endpoints del módulo en el ServeMux.
func (h *UMHandler) RegisterRoutes(mux *http.ServeMux, jwtMiddleware func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return jwtMiddleware(fn) }

	// Escritura — solo admin
	mux.Handle("POST /api/v1/unidades_medida", wrap(h.Crear))
	mux.Handle("PUT /api/v1/unidades_medida/{id}", wrap(h.Editar))
	mux.Handle("PATCH /api/v1/unidades_medida/{id}/inactivar", wrap(h.Inactivar))
	mux.Handle("GET /api/v1/unidades_medida/{id}/impacto", wrap(h.ObtenerImpacto))

	// Lectura — todos los roles autenticados
	mux.Handle("GET /api/v1/unidades_medida", wrap(h.Listar))
	mux.Handle("GET /api/v1/unidades_medida/{id}", wrap(h.ObtenerPorID))
}

// --- Handlers ---

// Crear maneja POST /api/v1/unidades_medida.
func (h *UMHandler) Crear(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "um.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/unidades_medida")),
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
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede gestionar unidades de medida."})
		return
	}

	var req CrearUMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}
	if req.Codigo == "" || req.Nombre == "" || req.TipoMedida == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "Los campos codigo, nombre y tipo_medida son obligatorios."})
		return
	}

	u, err := h.svc.Crear(&req, userID, claims.Rol)
	if err != nil {
		status, body := mapServiceError(err)
		span.SetStatus(codes.Error, err.Error())
		logOp(userID, claims.Rol, "crear_unidad", 0, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("unidad.id", int64(u.ID)))
	logOp(userID, claims.Rol, "crear_unidad", u.ID, http.StatusCreated, time.Since(inicio))
	h.recordMetrics(ctx, "crear_unidad", http.StatusCreated, durMs)
	writeJSON(w, http.StatusCreated, u)
}

// Listar maneja GET /api/v1/unidades_medida.
func (h *UMHandler) Listar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "um.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/unidades_medida")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}

	tipo := r.URL.Query().Get("tipo")
	if tipo != "" && !tiposValidos[tipo] {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "tipo_invalido", Mensaje: "El tipo debe ser peso, volumen o unidad.", Campo: "tipo"})
		return
	}

	params := &ListarUMParams{
		Tipo:  tipo,
		Page:  parseIntQuery(r, "page", 1),
		Limit: parseIntQuery(r, "limit", 50),
	}
	if params.Limit > 200 {
		params.Limit = 200
	}
	if estado := r.URL.Query().Get("estado"); estado != "" {
		estadosValidos := map[string]bool{"activo": true, "inactivo": true, "todos": true}
		if !estadosValidos[estado] {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "estado_invalido", Mensaje: "El estado debe ser 'activo', 'inactivo' o 'todos'.", Campo: "estado"})
			return
		}
		if estado != "todos" {
			b := estado == "activo"
			params.Activo = &b
		}
	}

	resp, err := h.svc.Listar(params)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."})
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logOp(userID, claims.Rol, "listar_unidades", 0, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "listar_unidades", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerPorID maneja GET /api/v1/unidades_medida/{id}.
func (h *UMHandler) ObtenerPorID(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "um.obtener",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/unidades_medida/{id}")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
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

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("unidad.id", int64(id)))
	logOp(userID, claims.Rol, "obtener_unidad", id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "obtener_unidad", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// Editar maneja PUT /api/v1/unidades_medida/{id}.
func (h *UMHandler) Editar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "um.editar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/unidades_medida/{id}")),
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
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede editar unidades de medida."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req EditarUMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}
	if req.Nombre == nil && req.FactorConversion == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "Debe enviar al menos un campo a editar."})
		return
	}

	u, svcErr := h.svc.Editar(id, &req, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "editar_unidad", id, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("unidad.id", int64(id)))
	logOp(userID, claims.Rol, "editar_unidad", id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "editar_unidad", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, u)
}

// Inactivar maneja PATCH /api/v1/unidades_medida/{id}/inactivar.
func (h *UMHandler) Inactivar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "um.inactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/unidades_medida/{id}/inactivar")),
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
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede inactivar unidades de medida."})
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
		logOp(userID, claims.Rol, "inactivar_unidad", id, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("unidad.id", int64(id)))
	logOp(userID, claims.Rol, "inactivar_unidad", id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "inactivar_unidad", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerImpacto maneja GET /api/v1/unidades_medida/{id}/impacto.
func (h *UMHandler) ObtenerImpacto(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "um.impacto",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/unidades_medida/{id}/impacto")),
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
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "acceso_denegado", Mensaje: "Solo el administrador puede consultar el impacto."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.ObtenerImpacto(id)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("unidad.id", int64(id)))
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

func logOp(userID uint64, rol, operacion string, unidadID uint64, statusHTTP int, dur time.Duration) {
	log.Printf(
		`{"user_id":%d,"rol":"%s","operacion":"%s","unidad_id":%d,"status_http":%d,"duracion_ms":%d}`,
		userID, rol, operacion, unidadID, statusHTTP, dur.Milliseconds(),
	)
}

func (h *UMHandler) recordMetrics(ctx context.Context, operacion string, status int, durMs float64) {
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
	switch {
	case errors.Is(err, ErrUnidadNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "unidad_no_encontrada", Mensaje: "La unidad de medida no existe."}
	case errors.Is(err, ErrCodigoDuplicado):
		return http.StatusConflict, errorResponse{Error: "codigo_duplicado", Mensaje: "Ya existe una unidad con ese código.", Campo: "codigo"}
	case errors.Is(err, ErrTipoInvalido):
		return http.StatusBadRequest, errorResponse{Error: "tipo_invalido", Mensaje: "El tipo de medida debe ser peso, volumen o unidad.", Campo: "tipo_medida"}
	case errors.Is(err, ErrFactorInvalido):
		return http.StatusUnprocessableEntity, errorResponse{Error: "factor_invalido", Mensaje: "El factor de conversión debe ser mayor que cero.", Campo: "factor_conversion"}
	case errors.Is(err, ErrYaInactiva):
		return http.StatusConflict, errorResponse{Error: "ya_inactiva", Mensaje: "La unidad ya está inactiva."}
	case errors.Is(err, ErrUnidadBaseNoInactivable):
		return http.StatusUnprocessableEntity, errorResponse{Error: "unidad_base_no_inactivable", Mensaje: "No se puede inactivar la unidad base mientras existan otras unidades activas del mismo tipo."}
	case errors.Is(err, ErrFactorBaseInmutable):
		return http.StatusUnprocessableEntity, errorResponse{Error: "factor_base_inmutable", Mensaje: "El factor de conversión de la unidad base no puede modificarse."}
	default:
		return http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."}
	}
}
