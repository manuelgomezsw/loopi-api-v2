package inventarios

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"go.opentelemetry.io/otel"
)

// Handler maneja los requests HTTP para inventarios
type Handler struct {
	service Service
	logger  *slog.Logger
	tracer  interface{} // Placeholder para futura integración con otel.Tracer
}

// NewHandler crea una nueva instancia del handler
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
		logger:  slog.Default(),
		tracer:  otel.Tracer("inventarios"),
	}
}

// GetSugerencia retorna la sugerencia de tipo/horario basada en la hora actual
// GET /api/v1/inventarios/sugerencia
func (h *Handler) GetSugerencia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	h.logger.InfoContext(ctx, "inventario.sugerencia.get: iniciando")

	sugerencia, err := h.service.Sugerir(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.sugerencia.get: error",
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.sugerencia.get: success",
		"tipo", sugerencia.Tipo,
		"horario", sugerencia.Horario)
	h.respondJSON(w, http.StatusOK, sugerencia)
}

// PostInventario inicia un nuevo conteo
// POST /api/v1/inventarios
func (h *Handler) PostInventario(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateInventarioReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(ctx, "inventario.iniciar.post: decode error",
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.iniciar.post: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.iniciar.post: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.iniciar.post: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"tienda_id", req.TiendaID,
		"tipo", req.Tipo,
		"horario", req.Horario)

	resp, err := h.service.Iniciar(ctx, &req, userID, claims.Rol)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.iniciar.post: error",
			"error", err.Error(),
			"tienda_id", req.TiendaID)
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.iniciar.post: success",
		"inventario_id", resp.ID,
		"tienda_id", resp.TiendaID,
		"items_count", len(resp.Items))
	h.respondJSON(w, http.StatusCreated, resp)
}

// GetInventario obtiene un inventario existente
// GET /api/v1/inventarios/{id}
func (h *Handler) GetInventario(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	if idStr == "" {
		h.logger.WarnContext(ctx, "inventario.detalle.get: id missing")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id requerido")
		return
	}

	var inventarioID int64
	if _, err := parseID(idStr, &inventarioID); err != nil {
		h.logger.WarnContext(ctx, "inventario.detalle.get: invalid id",
			"id_str", idStr,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id inválido")
		return
	}

	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.detalle.get: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.detalle.get: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.detalle.get: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"inventario_id", inventarioID)

	resp, err := h.service.Buscar(ctx, inventarioID, userID, claims.Rol)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.detalle.get: not found",
			"inventario_id", inventarioID,
			"error", err.Error())
		h.respondError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.detalle.get: success",
		"inventario_id", resp.ID,
		"estado", resp.Estado)
	h.respondJSON(w, http.StatusOK, resp)
}

// PatchItemValor registra el valor real de un item
// PATCH /api/v1/inventarios/{id}/items/{item_id}
func (h *Handler) PatchItemValor(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req PatchItemValorReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "inventario.registrar.patch: decode error",
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	inventarioIDStr := r.PathValue("id")
	if inventarioIDStr == "" {
		h.logger.WarnContext(ctx, "inventario.registrar.patch: inventario id missing")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id requerido")
		return
	}

	var inventarioID int64
	if _, err := parseID(inventarioIDStr, &inventarioID); err != nil {
		h.logger.WarnContext(ctx, "inventario.registrar.patch: invalid inventario id",
			"id_str", inventarioIDStr,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id inválido")
		return
	}

	itemIDStr := r.PathValue("item_id")
	if itemIDStr == "" {
		h.logger.WarnContext(ctx, "inventario.registrar.patch: item id missing")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "item id requerido")
		return
	}

	var itemID int64
	if _, err := parseID(itemIDStr, &itemID); err != nil {
		h.logger.WarnContext(ctx, "inventario.registrar.patch: invalid item id",
			"id_str", itemIDStr,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", "item id inválido")
		return
	}

	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.registrar.patch: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.registrar.patch: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.registrar.patch: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"inventario_id", inventarioID,
		"item_id", itemID,
		"valor_real", req.ValorReal)

	resp, err := h.service.RegistrarValor(ctx, inventarioID, itemID, req.ValorReal, userID)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.registrar.patch: error",
			"inventario_id", inventarioID,
			"item_id", itemID,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.registrar.patch: success",
		"inventario_id", inventarioID,
		"item_id", itemID,
		"diferencia", resp.Diferencia)
	h.respondJSON(w, http.StatusOK, resp)
}

// PostConfirmar marca un conteo como completado
// POST /api/v1/inventarios/{id}/confirmar
func (h *Handler) PostConfirmar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inventarioIDStr := r.PathValue("id")
	if inventarioIDStr == "" {
		h.logger.WarnContext(ctx, "inventario.confirmar.post: inventario id missing")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id requerido")
		return
	}

	var inventarioID int64
	if _, err := parseID(inventarioIDStr, &inventarioID); err != nil {
		h.logger.WarnContext(ctx, "inventario.confirmar.post: invalid inventario id",
			"id_str", inventarioIDStr,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id inválido")
		return
	}

	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.confirmar.post: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.confirmar.post: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.confirmar.post: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"inventario_id", inventarioID)

	resp, err := h.service.Confirmar(ctx, inventarioID, userID)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.confirmar.post: error",
			"inventario_id", inventarioID,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.confirmar.post: success",
		"inventario_id", resp.ID,
		"estado", resp.Estado,
		"completado_en", resp.CompletadoEn)
	h.respondJSON(w, http.StatusOK, resp)
}

// DeleteInventario elimina un conteo en progreso
// DELETE /api/v1/inventarios/{id}
func (h *Handler) DeleteInventario(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	inventarioIDStr := r.PathValue("id")
	if inventarioIDStr == "" {
		h.logger.WarnContext(ctx, "inventario.eliminar.delete: inventario id missing")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id requerido")
		return
	}

	var inventarioID int64
	if _, err := parseID(inventarioIDStr, &inventarioID); err != nil {
		h.logger.WarnContext(ctx, "inventario.eliminar.delete: invalid inventario id",
			"id_str", inventarioIDStr,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id inválido")
		return
	}

	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.eliminar.delete: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.eliminar.delete: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.eliminar.delete: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"inventario_id", inventarioID)

	err = h.service.Eliminar(ctx, inventarioID, userID, claims.Rol)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.eliminar.delete: error",
			"inventario_id", inventarioID,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.eliminar.delete: success",
		"inventario_id", inventarioID)
	w.WriteHeader(http.StatusNoContent)
}

// GetHistorial obtiene el listado de conteos con filtros y paginación
// GET /api/v1/inventarios
func (h *Handler) GetHistorial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filtros := &FiltrosInventario{
		Pagina:    1,
		PorPagina: 50,
	}

	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.historial.get: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.historial.get: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.historial.get: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"pagina", filtros.Pagina,
		"por_pagina", filtros.PorPagina)

	resp, err := h.service.Listar(ctx, filtros, userID, claims.Rol)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.historial.get: error",
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.historial.get: success",
		"total", resp.Total,
		"pagina", filtros.Pagina,
		"items_returned", len(resp.Inventarios))
	h.respondJSON(w, http.StatusOK, resp)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResp{
		Error:   errCode,
		Mensaje: message,
	})
}

func parseID(s string, id *int64) (bool, error) {
	parsed, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return false, err
	}
	*id = parsed
	return true, nil
}

// RegisterRoutes registra todas las rutas del módulo de inventarios en el multiplexor HTTP.
// Aplica jwtMiddleware donde es requerido per API contracts.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, middleware func(http.Handler) http.Handler) {
	// GET /api/v1/inventarios/sugerencia — público (no requiere autenticación)
	mux.HandleFunc("GET /api/v1/inventarios/sugerencia", h.GetSugerencia)

	// POST /api/v1/inventarios — requiere autenticación (T029)
	mux.Handle("POST /api/v1/inventarios", middleware(http.HandlerFunc(h.PostInventario)))

	// GET /api/v1/inventarios/{id} — requiere autenticación (detalle y historial, endpoint dual)
	mux.Handle("GET /api/v1/inventarios/{id}", middleware(http.HandlerFunc(h.GetInventario)))

	// PATCH /api/v1/inventarios/{id}/items/{item_id} — requiere autenticación
	mux.Handle("PATCH /api/v1/inventarios/{id}/items/{item_id}", middleware(http.HandlerFunc(h.PatchItemValor)))

	// POST /api/v1/inventarios/{id}/confirmar — requiere autenticación
	mux.Handle("POST /api/v1/inventarios/{id}/confirmar", middleware(http.HandlerFunc(h.PostConfirmar)))

	// DELETE /api/v1/inventarios/{id} — requiere autenticación (admin solo)
	mux.Handle("DELETE /api/v1/inventarios/{id}", middleware(http.HandlerFunc(h.DeleteInventario)))

	// GET /api/v1/inventarios — requiere autenticación (historial con filtros y paginación)
	mux.Handle("GET /api/v1/inventarios", middleware(http.HandlerFunc(h.GetHistorial)))
}
