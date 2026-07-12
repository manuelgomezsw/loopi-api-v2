package items

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

const otelScope = "loopi-api/items"

var tracer = otel.Tracer(otelScope)

// Metrics agrupa los instrumentos OTel del módulo items.
type Metrics struct {
	CreacionDuration      metric.Float64Histogram
	CreacionTotal         metric.Int64Counter
	ActualizacionDuration metric.Float64Histogram
	ActualizacionTotal    metric.Int64Counter
	CostosTiendaTotal     metric.Int64Counter
	ListadoDuration       metric.Float64Histogram
}

// NewMetrics inicializa los instrumentos OTel del módulo.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(otelScope)
	creacionDur, err := meter.Float64Histogram("items.creacion.duration",
		metric.WithDescription("Duración de creación de items en milisegundos"), metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	creacionTotal, err := meter.Int64Counter("items.creacion.total",
		metric.WithDescription("Conteo de creaciones de items por resultado"))
	if err != nil {
		return nil, err
	}
	actDur, err := meter.Float64Histogram("items.actualizacion.duration",
		metric.WithDescription("Duración de actualización de items en milisegundos"), metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	actTotal, err := meter.Int64Counter("items.actualizacion.total",
		metric.WithDescription("Conteo de actualizaciones de items por resultado"))
	if err != nil {
		return nil, err
	}
	costosTotal, err := meter.Int64Counter("items.costos_tienda.registro.total",
		metric.WithDescription("Conteo de registros de costo por tienda"))
	if err != nil {
		return nil, err
	}
	listadoDur, err := meter.Float64Histogram("items.listado.duration",
		metric.WithDescription("Duración de listado de items en milisegundos"), metric.WithUnit("ms"))
	if err != nil {
		return nil, err
	}
	return &Metrics{
		CreacionDuration: creacionDur, CreacionTotal: creacionTotal,
		ActualizacionDuration: actDur, ActualizacionTotal: actTotal,
		CostosTiendaTotal: costosTotal, ListadoDuration: listadoDur,
	}, nil
}

func noopMetrics() *Metrics {
	m, _ := NewMetrics()
	if m == nil {
		return &Metrics{}
	}
	return m
}

// Handler agrupa los handlers HTTP del módulo items.
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
// La lectura del catálogo (listado/detalle) está abierta a cualquier rol autenticado.
// La escritura y el historial de costos por tienda son exclusivos de admin.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, jwtMiddleware func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return jwtMiddleware(fn) }

	mux.Handle("GET /api/v1/items", wrap(h.Listar))
	mux.Handle("GET /api/v1/items/{id}", wrap(h.ObtenerPorID))
	mux.Handle("POST /api/v1/items", wrap(h.Crear))
	mux.Handle("PUT /api/v1/items/{id}", wrap(h.Editar))
	mux.Handle("PATCH /api/v1/items/{id}/inactivar", wrap(h.Inactivar))
	mux.Handle("PATCH /api/v1/items/{id}/reactivar", wrap(h.Reactivar))
	mux.Handle("GET /api/v1/items/{id}/costos_tienda", wrap(h.ListarCostosTienda))
	mux.Handle("POST /api/v1/items/{id}/costos_tienda", wrap(h.RegistrarCostoTienda))
}

// --- Handlers ---

// Crear maneja POST /api/v1/items.
func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "items.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/items")))
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede crear items."})
		return
	}

	var req CrearItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	it, err := h.svc.Crear(&req, userID, claims.Rol)
	if err != nil {
		status, body := mapServiceError(err)
		span.SetStatus(codes.Error, err.Error())
		h.recordCreacion(ctx, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("item.id", int64(it.ID)), attribute.String("item.codigo", it.Codigo))
	h.recordCreacion(ctx, http.StatusCreated, time.Since(inicio))
	writeJSON(w, http.StatusCreated, it)
}

// Listar maneja GET /api/v1/items.
func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "items.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/items")))
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	if _, ok := userIDFromClaims(claims); !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}

	filtros := &FiltrosListado{
		Pagina:    parseIntQuery(r, "pagina", 1),
		PorPagina: parseIntQuery(r, "por_pagina", 50),
	}
	if filtros.PorPagina > 200 {
		filtros.PorPagina = 200
	}
	if tipo := r.URL.Query().Get("tipo"); tipo != "" {
		if !tiposValidos[tipo] {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "parametro_invalido", Mensaje: "El tipo indicado no es válido.", Campo: "tipo"})
			return
		}
		filtros.Tipo = tipo
	}
	if frecuencia := r.URL.Query().Get("frecuencia"); frecuencia != "" {
		if !frecuenciasValidas[frecuencia] {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "parametro_invalido", Mensaje: "La frecuencia indicada no es válida.", Campo: "frecuencia"})
			return
		}
		filtros.Frecuencia = frecuencia
	}
	if activoStr := r.URL.Query().Get("activo"); activoStr != "" {
		activo, err := strconv.ParseBool(activoStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "parametro_invalido", Mensaje: "El valor de activo no es válido.", Campo: "activo"})
			return
		}
		filtros.Activo = &activo
	}

	resp, err := h.svc.Listar(filtros)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."})
		return
	}

	if h.metrics != nil && h.metrics.ListadoDuration != nil {
		h.metrics.ListadoDuration.Record(ctx, float64(time.Since(inicio).Milliseconds()))
	}
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerPorID maneja GET /api/v1/items/{id}.
func (h *Handler) ObtenerPorID(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "items.obtener",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/items/{id}")))
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	if _, ok := userIDFromClaims(claims); !ok {
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

	span.SetAttributes(attribute.Int64("item.id", int64(id)))
	writeJSON(w, http.StatusOK, resp)
}

// Editar maneja PUT /api/v1/items/{id}.
func (h *Handler) Editar(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "items.actualizar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/items/{id}")))
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede editar items."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req EditarItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	it, svcErr := h.svc.Editar(id, &req, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		h.recordActualizacion(ctx, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("item.id", int64(id)))
	h.recordActualizacion(ctx, http.StatusOK, time.Since(inicio))
	writeJSON(w, http.StatusOK, it)
}

// Inactivar maneja PATCH /api/v1/items/{id}/inactivar.
func (h *Handler) Inactivar(w http.ResponseWriter, r *http.Request) {
	h.cambiarEstado(w, r, "inactivar", h.svc.Inactivar)
}

// Reactivar maneja PATCH /api/v1/items/{id}/reactivar.
func (h *Handler) Reactivar(w http.ResponseWriter, r *http.Request) {
	h.cambiarEstado(w, r, "reactivar", h.svc.Reactivar)
}

func (h *Handler) cambiarEstado(w http.ResponseWriter, r *http.Request, operacion string, accion func(id, userID uint64, rol string) (*CambiarEstadoResponse, error)) {
	ctx, span := tracer.Start(r.Context(), "items.cambiar_estado",
		trace.WithAttributes(
			attribute.String("http.route", fmt.Sprintf("/api/v1/items/{id}/%s", operacion)),
			attribute.String("operacion", operacion),
		))
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede cambiar el estado de un item."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := accion(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("item.id", int64(id)))
	writeJSON(w, http.StatusOK, resp)
}

// RegistrarCostoTienda maneja POST /api/v1/items/{id}/costos_tienda.
func (h *Handler) RegistrarCostoTienda(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "items.costos_tienda.registrar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/items/{id}/costos_tienda")))
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede gestionar costos por tienda."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req RegistrarCostoTiendaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "campo_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}

	c, svcErr := h.svc.RegistrarCostoTienda(id, &req, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		if h.metrics != nil && h.metrics.CostosTiendaTotal != nil {
			h.metrics.CostosTiendaTotal.Add(ctx, 1, metric.WithAttributes(attribute.Int("status_http", status), attribute.Int64("tienda_id", int64(req.TiendaID))))
		}
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("item.id", int64(id)), attribute.Int64("tienda_id", int64(req.TiendaID)))
	if h.metrics != nil && h.metrics.CostosTiendaTotal != nil {
		h.metrics.CostosTiendaTotal.Add(ctx, 1, metric.WithAttributes(attribute.Int("status_http", http.StatusCreated), attribute.Int64("tienda_id", int64(req.TiendaID))))
	}
	writeJSON(w, http.StatusCreated, c)
}

// ListarCostosTienda maneja GET /api/v1/items/{id}/costos_tienda.
func (h *Handler) ListarCostosTienda(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "items.costos_tienda.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/items/{id}/costos_tienda")))
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	if _, ok := userIDFromClaims(claims); !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}
	if claims.Rol != "admin" {
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede consultar el historial de costos por tienda."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.ObtenerHistorialCostos(id)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("item.id", int64(id)))
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

func (h *Handler) recordCreacion(ctx context.Context, status int, dur time.Duration) {
	if h.metrics == nil {
		return
	}
	attrs := metric.WithAttributes(attribute.Int("status_http", status))
	if h.metrics.CreacionDuration != nil {
		h.metrics.CreacionDuration.Record(ctx, float64(dur.Milliseconds()), attrs)
	}
	if h.metrics.CreacionTotal != nil {
		h.metrics.CreacionTotal.Add(ctx, 1, attrs)
	}
}

func (h *Handler) recordActualizacion(ctx context.Context, status int, dur time.Duration) {
	if h.metrics == nil {
		return
	}
	attrs := metric.WithAttributes(attribute.Int("status_http", status))
	if h.metrics.ActualizacionDuration != nil {
		h.metrics.ActualizacionDuration.Record(ctx, float64(dur.Milliseconds()), attrs)
	}
	if h.metrics.ActualizacionTotal != nil {
		h.metrics.ActualizacionTotal.Add(ctx, 1, attrs)
	}
}

func mapServiceError(err error) (int, errorResponse) {
	var valErr *ValidationError
	switch {
	case errors.As(err, &valErr):
		return http.StatusBadRequest, errorResponse{Error: valErr.Codigo, Mensaje: valErr.Mensaje, Campo: valErr.Campo}
	case errors.Is(err, ErrItemNoEncontrado):
		return http.StatusNotFound, errorResponse{Error: "item_no_encontrado", Mensaje: "El item no existe."}
	case errors.Is(err, ErrCodigoDuplicado):
		return http.StatusConflict, errorResponse{Error: "codigo_duplicado", Mensaje: "Ya existe un item con ese código.", Campo: "codigo"}
	case errors.Is(err, ErrNombreDuplicado):
		return http.StatusConflict, errorResponse{Error: "nombre_duplicado", Mensaje: "Ya existe un item con ese nombre.", Campo: "nombre"}
	case errors.Is(err, ErrCodigoEnUso):
		return http.StatusUnprocessableEntity, errorResponse{Error: "codigo_en_uso", Mensaje: "El item ya tiene usos registrados; el código no se puede modificar.", Campo: "codigo"}
	case errors.Is(err, ErrCambioUnidadRequiereConfirmacion):
		return http.StatusUnprocessableEntity, errorResponse{Error: "cambio_unidad_requiere_confirmacion", Mensaje: "El item tiene historial de stock; confirma el cambio de unidad de medida.", Campo: "unidad_medida_id"}
	case errors.Is(err, ErrSubcategoriaNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "subcategoria_no_encontrada", Mensaje: "La subcategoría indicada no existe.", Campo: "subcategoria_id"}
	case errors.Is(err, ErrSubcategoriaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "subcategoria_inactiva", Mensaje: "La subcategoría indicada está inactiva.", Campo: "subcategoria_id"}
	case errors.Is(err, ErrProveedorNoEncontrado):
		return http.StatusNotFound, errorResponse{Error: "proveedor_no_encontrado", Mensaje: "El proveedor indicado no existe.", Campo: "proveedor_id"}
	case errors.Is(err, ErrProveedorInactivo):
		return http.StatusUnprocessableEntity, errorResponse{Error: "proveedor_inactivo", Mensaje: "El proveedor indicado está inactivo.", Campo: "proveedor_id"}
	case errors.Is(err, ErrUnidadMedidaNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "unidad_medida_no_encontrada", Mensaje: "La unidad de medida indicada no existe.", Campo: "unidad_medida_id"}
	case errors.Is(err, ErrUnidadMedidaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "unidad_medida_inactiva", Mensaje: "La unidad de medida indicada está inactiva.", Campo: "unidad_medida_id"}
	case errors.Is(err, ErrTiendaNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "tienda_no_encontrada", Mensaje: "La tienda indicada no existe.", Campo: "tienda_id"}
	case errors.Is(err, ErrTiendaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "tienda_inactiva", Mensaje: "La tienda indicada está inactiva.", Campo: "tienda_id"}
	case errors.Is(err, ErrItemYaInactivo):
		return http.StatusUnprocessableEntity, errorResponse{Error: "item_ya_inactivo", Mensaje: "El item ya estaba inactivo."}
	case errors.Is(err, ErrItemYaActivo):
		return http.StatusUnprocessableEntity, errorResponse{Error: "item_ya_activo", Mensaje: "El item ya estaba activo."}
	default:
		log.Printf(`{"level":"error","msg":"error interno en items","error":%q}`, err.Error())
		return http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."}
	}
}
