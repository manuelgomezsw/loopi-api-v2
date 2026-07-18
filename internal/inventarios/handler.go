package inventarios

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

// intPtrToInt64Ptr convierte *int a *int64
func intPtrToInt64Ptr(p *int) *int64 {
	if p == nil {
		return nil
	}
	v := int64(*p)
	return &v
}

// mapErrorToStatus mapea códigos de error del service a status HTTP per contracts/api.md
func mapErrorToStatus(errCode string) int {
	switch errCode {
	// 201 Created — handled in handlers explicitly
	// 204 No Content — handled in handlers explicitly

	// 400 Bad Request
	case "sin_tienda", "validation_error", "invalid_request":
		return http.StatusBadRequest

	// 401 Unauthorized
	case "unauthorized":
		return http.StatusUnauthorized

	// 403 Forbidden
	case "tienda_no_autorizada", "sin_permiso", "conteo_bloqueado":
		return http.StatusForbidden

	// 404 Not Found
	case "not_found":
		return http.StatusNotFound

	// 409 Conflict
	case "conteo_duplicado", "ya_completado":
		return http.StatusConflict

	// 422 Unprocessable Entity
	case "items_sin_registrar", "estado_invalido", "eliminacion_no_permitida", "sin_items_contabilizar":
		return http.StatusUnprocessableEntity

	// Default: 400 Bad Request
	default:
		return http.StatusBadRequest
	}
}


// GetSugerencia retorna la sugerencia de tipo/horario basada en la hora actual
// GET /api/v1/inventarios/sugerencia
func (h *Handler) GetSugerencia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	userID, _ := strconv.ParseInt(claims.Subject, 10, 64)

	h.logger.InfoContext(ctx, "inventario.sugerencia.get: iniciando",
		"user_id", userID,
		"rol", claims.Rol,
		"tienda_id", claims.TiendaID)

	sugerencia, err := h.service.Sugerir(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.sugerencia.get: error",
			"user_id", userID,
			"rol", claims.Rol,
			"tienda_id", claims.TiendaID,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.sugerencia.get: success",
		"user_id", userID,
		"rol", claims.Rol,
		"tienda_id", claims.TiendaID,
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
		"user_tienda_id", claims.TiendaID,
		"tienda_id", req.TiendaID,
		"tipo", req.Tipo,
		"horario", req.Horario)

	resp, err := h.service.Iniciar(ctx, &req, userID, claims.Rol, intPtrToInt64Ptr(claims.TiendaID))
	if err != nil {
		svcErr, ok := err.(*Error)
		if !ok {
			h.logger.ErrorContext(ctx, "inventario.iniciar.post: error",
				"user_id", userID,
				"rol", claims.Rol,
				"error", err.Error(),
				"tienda_id", req.TiendaID)
			h.respondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		h.logger.WarnContext(ctx, "inventario.iniciar.post: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error_code", svcErr.Code,
			"error_message", svcErr.Message,
			"tienda_id", req.TiendaID)
		h.respondError(w, mapErrorToStatus(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}

	h.logger.InfoContext(ctx, "inventario.iniciar.post: success",
		"user_id", userID,
		"rol", claims.Rol,
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
		"user_tienda_id", claims.TiendaID,
		"inventario_id", inventarioID)

	resp, err := h.service.Buscar(ctx, inventarioID, userID, claims.Rol, intPtrToInt64Ptr(claims.TiendaID))
	if err != nil {
		svcErr, ok := err.(*Error)
		if !ok {
			h.logger.ErrorContext(ctx, "inventario.detalle.get: error",
				"user_id", userID,
				"rol", claims.Rol,
				"inventario_id", inventarioID,
				"error", err.Error())
			h.respondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		h.logger.WarnContext(ctx, "inventario.detalle.get: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error_code", svcErr.Code,
			"inventario_id", inventarioID)
		h.respondError(w, mapErrorToStatus(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}

	h.logger.InfoContext(ctx, "inventario.detalle.get: success",
		"user_id", userID,
		"rol", claims.Rol,
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
		svcErr, ok := err.(*Error)
		if !ok {
			h.logger.ErrorContext(ctx, "inventario.registrar.patch: error",
				"user_id", userID,
				"rol", claims.Rol,
				"inventario_id", inventarioID,
				"item_id", itemID,
				"error", err.Error())
			h.respondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		h.logger.WarnContext(ctx, "inventario.registrar.patch: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error_code", svcErr.Code,
			"inventario_id", inventarioID,
			"item_id", itemID)
		h.respondError(w, mapErrorToStatus(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}

	h.logger.InfoContext(ctx, "inventario.registrar.patch: success",
		"user_id", userID,
		"rol", claims.Rol,
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
		svcErr, ok := err.(*Error)
		if !ok {
			h.logger.ErrorContext(ctx, "inventario.confirmar.post: error",
				"user_id", userID,
				"rol", claims.Rol,
				"inventario_id", inventarioID,
				"error", err.Error())
			h.respondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		h.logger.WarnContext(ctx, "inventario.confirmar.post: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error_code", svcErr.Code,
			"inventario_id", inventarioID)
		h.respondError(w, mapErrorToStatus(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}

	h.logger.InfoContext(ctx, "inventario.confirmar.post: success",
		"user_id", userID,
		"rol", claims.Rol,
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
		"user_tienda_id", claims.TiendaID,
		"inventario_id", inventarioID)

	err = h.service.Eliminar(ctx, inventarioID, userID, claims.Rol, intPtrToInt64Ptr(claims.TiendaID))
	if err != nil {
		svcErr, ok := err.(*Error)
		if !ok {
			h.logger.ErrorContext(ctx, "inventario.eliminar.delete: error",
				"user_id", userID,
				"rol", claims.Rol,
				"inventario_id", inventarioID,
				"error", err.Error())
			h.respondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		h.logger.WarnContext(ctx, "inventario.eliminar.delete: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error_code", svcErr.Code,
			"inventario_id", inventarioID)
		h.respondError(w, mapErrorToStatus(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}

	h.logger.InfoContext(ctx, "inventario.eliminar.delete: success",
		"user_id", userID,
		"rol", claims.Rol,
		"inventario_id", inventarioID)
	w.WriteHeader(http.StatusNoContent)
}

// parseHistorialParams parsea los query parameters y retorna un FiltrosInventario
func (h *Handler) parseHistorialParams(r *http.Request) *FiltrosInventario {
	q := r.URL.Query()

	filtros := &FiltrosInventario{
		Pagina:    1,
		PorPagina: 50,
	}

	// Parsear tienda_id
	if tiendaIDStr := q.Get("tienda_id"); tiendaIDStr != "" {
		if id, err := strconv.ParseInt(tiendaIDStr, 10, 64); err == nil {
			filtros.TiendaID = &id
		}
	}

	// Parsear tipo
	if tipoStr := q.Get("tipo"); tipoStr != "" {
		tiposValidos := []string{"diario", "semanal", "mensual", "inicial"}
		for _, v := range tiposValidos {
			if tipoStr == v {
				t := Tipo(tipoStr)
				filtros.Tipo = &t
				break
			}
		}
	}

	// Parsear estado
	if estadoStr := q.Get("estado"); estadoStr != "" {
		estadosValidos := []string{"en_progreso", "completado"}
		for _, v := range estadosValidos {
			if estadoStr == v {
				e := Estado(estadoStr)
				filtros.Estado = &e
				break
			}
		}
	}

	// Parsear fechas
	if desdeStr := q.Get("desde"); desdeStr != "" {
		if fecha, err := time.Parse("2006-01-02", desdeStr); err == nil {
			filtros.Desde = &fecha
		}
	}
	if hastaStr := q.Get("hasta"); hastaStr != "" {
		if fecha, err := time.Parse("2006-01-02", hastaStr); err == nil {
			filtros.Hasta = &fecha
		}
	}

	// Parsear pagina
	if paginaStr := q.Get("pagina"); paginaStr != "" {
		if pagina, err := strconv.Atoi(paginaStr); err == nil && pagina >= 1 {
			filtros.Pagina = pagina
		}
	}

	// Parsear por_pagina (máximo 200 per spec)
	if porPaginaStr := q.Get("por_pagina"); porPaginaStr != "" {
		if porPagina, err := strconv.Atoi(porPaginaStr); err == nil {
			if porPagina > 200 {
				porPagina = 200
			} else if porPagina < 1 {
				porPagina = 1
			}
			filtros.PorPagina = porPagina
		}
	}

	return filtros
}

// GetHistorial obtiene el listado de conteos con filtros y paginación
// GET /api/v1/inventarios
func (h *Handler) GetHistorial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filtros := h.parseHistorialParams(r)

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
		"user_tienda_id", claims.TiendaID,
		"pagina", filtros.Pagina,
		"por_pagina", filtros.PorPagina)

	resp, err := h.service.Listar(ctx, filtros, userID, claims.Rol, intPtrToInt64Ptr(claims.TiendaID))
	if err != nil {
		svcErr, ok := err.(*Error)
		if !ok {
			h.logger.ErrorContext(ctx, "inventario.historial.get: error",
				"user_id", userID,
				"rol", claims.Rol,
				"error", err.Error())
			h.respondError(w, http.StatusBadRequest, "error", err.Error())
			return
		}
		h.logger.WarnContext(ctx, "inventario.historial.get: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error_code", svcErr.Code)
		h.respondError(w, mapErrorToStatus(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}

	h.logger.InfoContext(ctx, "inventario.historial.get: success",
		"user_id", userID,
		"rol", claims.Rol,
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

// GetEstadoInventarioActivo verifica si hay un conteo activo en una tienda
// GET /api/v1/inventarios/estado?tienda_id=X
// T143: Utilizado por otros módulos para bloquear movimientos durante conteo activo
func (h *Handler) GetEstadoInventarioActivo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(auth.ContextKeyClaims).(*auth.Claims)
	userID, _ := strconv.ParseInt(claims.Subject, 10, 64)

	tiendaIDStr := r.URL.Query().Get("tienda_id")
	if tiendaIDStr == "" {
		h.logger.WarnContext(ctx, "inventario.estado.get: missing tienda_id parameter",
			"user_id", userID)
		h.respondError(w, http.StatusBadRequest, "validation_error", "tienda_id parameter es requerido")
		return
	}

	tiendaID, err := strconv.ParseInt(tiendaIDStr, 10, 64)
	if err != nil {
		h.logger.WarnContext(ctx, "inventario.estado.get: invalid tienda_id",
			"user_id", userID,
			"tienda_id", tiendaIDStr)
		h.respondError(w, http.StatusBadRequest, "validation_error", "tienda_id debe ser un número válido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.estado.get: iniciando",
		"user_id", userID,
		"tienda_id", tiendaID)

	// Verificar si hay conteo activo
	inv, err := h.service.GetEstadoInventarioActivo(ctx, tiendaID)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.estado.get: error verificando",
			"user_id", userID,
			"tienda_id", tiendaID,
			"error", err.Error())
		h.respondError(w, http.StatusInternalServerError, "error", "Error verificando estado del inventario")
		return
	}

	resp := EstadoInventarioResp{
		Activo:     inv != nil,
		Inventario: inv,
	}

	h.logger.InfoContext(ctx, "inventario.estado.get: success",
		"user_id", userID,
		"tienda_id", tiendaID,
		"activo", resp.Activo)

	h.respondJSON(w, http.StatusOK, resp)
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
	// GET /api/v1/inventarios/sugerencia — requiere autenticación (solo actores de feature)
	mux.Handle("GET /api/v1/inventarios/sugerencia", middleware(http.HandlerFunc(h.GetSugerencia)))

	// GET /api/v1/inventarios/estado — requiere autenticación (T143: verificar conteo activo)
	mux.Handle("GET /api/v1/inventarios/estado", middleware(http.HandlerFunc(h.GetEstadoInventarioActivo)))

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
