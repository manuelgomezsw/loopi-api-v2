package categorias

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

const otelScope = "loopi-api/categorias"

var tracer = otel.Tracer(otelScope)

// Metrics agrupa los instrumentos OTel del módulo categorias.
type Metrics struct {
	RequestDuration metric.Float64Histogram
	RequestCount    metric.Int64Counter
}

// NewMetrics inicializa los instrumentos OTel del módulo.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter(otelScope)
	dur, err := meter.Float64Histogram(
		"categorias.request.duration",
		metric.WithDescription("Duración de requests de categorias en milisegundos"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}
	cnt, err := meter.Int64Counter(
		"categorias.request.total",
		metric.WithDescription("Conteo de requests de categorias por operación y resultado"),
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

// Handler agrupa los handlers HTTP del módulo categorias.
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
func (h *Handler) RegisterRoutes(mux *http.ServeMux, jwtMiddleware func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return jwtMiddleware(fn) }

	// Lectura — todos los roles autenticados
	mux.Handle("GET /api/v1/categorias", wrap(h.ListarCatalogo))
	mux.Handle("GET /api/v1/categorias/{id}", wrap(h.ObtenerCategoria))
	mux.Handle("GET /api/v1/categorias/{id}/impacto", wrap(h.ObtenerImpacto))

	// Escritura — solo admin
	mux.Handle("POST /api/v1/categorias", wrap(h.CrearCategoria))
	mux.Handle("PUT /api/v1/categorias/{id}", wrap(h.EditarCategoria))
	mux.Handle("PATCH /api/v1/categorias/{id}/inactivar", wrap(h.InactivarCategoria))
	mux.Handle("PATCH /api/v1/categorias/{id}/reactivar", wrap(h.ReactivarCategoria))

	// Subcategorías — escritura — solo admin
	mux.Handle("POST /api/v1/subcategorias", wrap(h.CrearSubcategoria))
	mux.Handle("PUT /api/v1/subcategorias/{id}", wrap(h.EditarSubcategoria))
	mux.Handle("PATCH /api/v1/subcategorias/{id}/inactivar", wrap(h.InactivarSubcategoria))
	mux.Handle("PATCH /api/v1/subcategorias/{id}/reactivar", wrap(h.ReactivarSubcategoria))
}

// --- Handlers de categorías ---

// ListarCatalogo maneja GET /api/v1/categorias.
func (h *Handler) ListarCatalogo(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.listar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias")),
	)
	defer span.End()
	inicio := time.Now()

	claims := auth.ClaimsFromContext(ctx)
	userID, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}

	estado := r.URL.Query().Get("estado")
	if estado != "" && estado != "activo" && estado != "inactivo" && estado != "todos" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "estado_invalido", Mensaje: "El estado debe ser 'activo', 'inactivo' o 'todos'.", Campo: "estado"})
		return
	}

	resp, err := h.svc.ObtenerCatalogo(estado)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."})
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	logOp(userID, claims.Rol, "listar_catalogo", 0, 0, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "listar_catalogo", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerCategoria maneja GET /api/v1/categorias/{id}.
func (h *Handler) ObtenerCategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.obtener",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias/{id}")),
	)
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	_, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.ObtenerCategoria(id)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("categoria.id", int64(id)))
	writeJSON(w, http.StatusOK, resp)
}

// ObtenerImpacto maneja GET /api/v1/categorias/{id}/impacto.
func (h *Handler) ObtenerImpacto(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.impacto",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias/{id}/impacto")),
	)
	defer span.End()

	claims := auth.ClaimsFromContext(ctx)
	_, ok := userIDFromClaims(claims)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "no_autenticado", Mensaje: "Debes iniciar sesión."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.ObtenerImpactoCategoria(id)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		writeJSON(w, status, body)
		return
	}

	span.SetAttributes(attribute.Int64("categoria.id", int64(id)))
	writeJSON(w, http.StatusOK, resp)
}

// CrearCategoria maneja POST /api/v1/categorias.
func (h *Handler) CrearCategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede gestionar categorías."})
		return
	}

	var req struct{ Nombre string `json:"nombre"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Nombre == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "nombre_requerido", Mensaje: "El campo nombre es obligatorio.", Campo: "nombre"})
		return
	}

	resp, svcErr := h.svc.CrearCategoria(req.Nombre, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "crear_categoria", 0, 0, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("categoria.id", int64(resp.ID)), attribute.String("user.rol", claims.Rol))
	logOp(userID, claims.Rol, "crear_categoria", resp.ID, 0, http.StatusCreated, time.Since(inicio))
	h.recordMetrics(ctx, "crear_categoria", http.StatusCreated, durMs)
	writeJSON(w, http.StatusCreated, resp)
}

// EditarCategoria maneja PUT /api/v1/categorias/{id}.
func (h *Handler) EditarCategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.editar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias/{id}")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede editar categorías."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req struct{ Nombre string `json:"nombre"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Nombre == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "nombre_requerido", Mensaje: "El campo nombre es obligatorio.", Campo: "nombre"})
		return
	}

	c, svcErr := h.svc.EditarCategoria(id, req.Nombre, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "editar_categoria", 0, id, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("categoria.id", int64(id)), attribute.String("user.rol", claims.Rol))
	logOp(userID, claims.Rol, "editar_categoria", 0, id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "editar_categoria", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, c)
}

// InactivarCategoria maneja PATCH /api/v1/categorias/{id}/inactivar.
func (h *Handler) InactivarCategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.inactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias/{id}/inactivar")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede inactivar categorías."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.InactivarCategoria(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "inactivar_categoria", 0, id, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("categoria.id", int64(id)), attribute.String("user.rol", claims.Rol), attribute.String("operacion", "inactivar"))
	logOp(userID, claims.Rol, "inactivar_categoria", 0, id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "inactivar_categoria", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ReactivarCategoria maneja PATCH /api/v1/categorias/{id}/reactivar.
func (h *Handler) ReactivarCategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "cat.reactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/categorias/{id}/reactivar")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede reactivar categorías."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	c, svcErr := h.svc.ReactivarCategoria(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "reactivar_categoria", 0, id, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("categoria.id", int64(id)), attribute.String("user.rol", claims.Rol), attribute.String("operacion", "reactivar"))
	logOp(userID, claims.Rol, "reactivar_categoria", 0, id, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "reactivar_categoria", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, c)
}

// --- Handlers de subcategorías ---

// CrearSubcategoria maneja POST /api/v1/subcategorias.
func (h *Handler) CrearSubcategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "subcat.crear",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/subcategorias")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede gestionar subcategorías."})
		return
	}

	var req struct {
		Nombre      string `json:"nombre"`
		CategoriaID uint64 `json:"categoria_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "nombre_requerido", Mensaje: "El cuerpo de la solicitud no es válido."})
		return
	}
	if req.Nombre == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "nombre_requerido", Mensaje: "El campo nombre es obligatorio.", Campo: "nombre"})
		return
	}
	if req.CategoriaID == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "categoria_requerida", Mensaje: "El campo categoria_id es obligatorio.", Campo: "categoria_id"})
		return
	}

	resp, svcErr := h.svc.CrearSubcategoria(req.Nombre, req.CategoriaID, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "crear_subcategoria", 0, req.CategoriaID, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(
		attribute.Int64("subcategoria.id", int64(resp.ID)),
		attribute.Int64("categoria.id", int64(req.CategoriaID)),
		attribute.String("user.rol", claims.Rol),
	)
	logOp(userID, claims.Rol, "crear_subcategoria", resp.ID, req.CategoriaID, http.StatusCreated, time.Since(inicio))
	h.recordMetrics(ctx, "crear_subcategoria", http.StatusCreated, durMs)
	writeJSON(w, http.StatusCreated, resp)
}

// EditarSubcategoria maneja PUT /api/v1/subcategorias/{id}.
func (h *Handler) EditarSubcategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "subcat.editar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/subcategorias/{id}")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede editar subcategorías."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	var req struct{ Nombre string `json:"nombre"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Nombre == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "nombre_requerido", Mensaje: "El campo nombre es obligatorio.", Campo: "nombre"})
		return
	}

	sub, svcErr := h.svc.EditarSubcategoria(id, req.Nombre, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "editar_subcategoria", id, 0, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(attribute.Int64("subcategoria.id", int64(id)), attribute.String("user.rol", claims.Rol))
	logOp(userID, claims.Rol, "editar_subcategoria", id, 0, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "editar_subcategoria", http.StatusOK, durMs)
	// Retornar SubcategoriaResponse con total_items=0 (no almacenamos items en este contexto)
	writeJSON(w, http.StatusOK, SubcategoriaResponse{Subcategoria: *sub, TotalItems: 0})
}

// InactivarSubcategoria maneja PATCH /api/v1/subcategorias/{id}/inactivar.
func (h *Handler) InactivarSubcategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "subcat.inactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/subcategorias/{id}/inactivar")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede inactivar subcategorías."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	resp, svcErr := h.svc.InactivarSubcategoria(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "inactivar_subcategoria", id, 0, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(
		attribute.Int64("subcategoria.id", int64(id)),
		attribute.Int64("categoria.id", int64(resp.CategoriaID)),
		attribute.String("user.rol", claims.Rol),
		attribute.String("operacion", "inactivar"),
	)
	logOp(userID, claims.Rol, "inactivar_subcategoria", id, resp.CategoriaID, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "inactivar_subcategoria", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, resp)
}

// ReactivarSubcategoria maneja PATCH /api/v1/subcategorias/{id}/reactivar.
func (h *Handler) ReactivarSubcategoria(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "subcat.reactivar",
		trace.WithAttributes(attribute.String("http.route", "/api/v1/subcategorias/{id}/reactivar")),
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
		span.SetStatus(codes.Error, "sin_permiso")
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "sin_permiso", Mensaje: "Solo el administrador puede reactivar subcategorías."})
		return
	}

	id, err := parseIDPath(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "input_invalido", Mensaje: "ID inválido."})
		return
	}

	sub, svcErr := h.svc.ReactivarSubcategoria(id, userID, claims.Rol)
	if svcErr != nil {
		status, body := mapServiceError(svcErr)
		span.SetStatus(codes.Error, svcErr.Error())
		logOp(userID, claims.Rol, "reactivar_subcategoria", id, 0, status, time.Since(inicio))
		writeJSON(w, status, body)
		return
	}

	durMs := float64(time.Since(inicio).Milliseconds())
	span.SetAttributes(
		attribute.Int64("subcategoria.id", int64(id)),
		attribute.Int64("categoria.id", int64(sub.CategoriaID)),
		attribute.String("user.rol", claims.Rol),
		attribute.String("operacion", "reactivar"),
	)
	logOp(userID, claims.Rol, "reactivar_subcategoria", id, sub.CategoriaID, http.StatusOK, time.Since(inicio))
	h.recordMetrics(ctx, "reactivar_subcategoria", http.StatusOK, durMs)
	writeJSON(w, http.StatusOK, sub)
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

func logOp(userID uint64, rol, operacion string, subcatID, catID uint64, statusHTTP int, dur time.Duration) {
	log.Printf(
		`{"user_id":%d,"rol":"%s","operacion":"%s","categoria_id":%d,"subcategoria_id":%d,"status_http":%d,"duracion_ms":%d}`,
		userID, rol, operacion, catID, subcatID, statusHTTP, dur.Milliseconds(),
	)
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
	switch {
	case errors.Is(err, ErrCategoriaNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "categoria_no_encontrada", Mensaje: "La categoría no existe."}
	case errors.Is(err, ErrSubcategoriaNoEncontrada):
		return http.StatusNotFound, errorResponse{Error: "subcategoria_no_encontrada", Mensaje: "La subcategoría no existe."}
	case errors.Is(err, ErrNombreDuplicado):
		return http.StatusConflict, errorResponse{Error: "nombre_duplicado", Mensaje: "Ya existe un registro con ese nombre.", Campo: "nombre"}
	case errors.Is(err, ErrCategoriaPadreInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "categoria_padre_inactiva", Mensaje: "La categoría padre está inactiva.", Campo: "categoria_id"}
	case errors.Is(err, ErrCategoriaYaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "categoria_ya_inactiva", Mensaje: "La categoría ya está inactiva."}
	case errors.Is(err, ErrCategoriaYaActiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "categoria_ya_activa", Mensaje: "La categoría ya está activa."}
	case errors.Is(err, ErrSubcategoriaYaInactiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "subcategoria_ya_inactiva", Mensaje: "La subcategoría ya está inactiva."}
	case errors.Is(err, ErrSubcategoriaYaActiva):
		return http.StatusUnprocessableEntity, errorResponse{Error: "subcategoria_ya_activa", Mensaje: "La subcategoría ya está activa."}
	default:
		return http.StatusInternalServerError, errorResponse{Error: "error_interno", Mensaje: "Error al procesar la solicitud."}
	}
}
