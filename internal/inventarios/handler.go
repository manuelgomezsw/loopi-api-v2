package inventarios

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Handler maneja los requests HTTP para inventarios
type Handler struct {
	service Service
	// TODO: Agregar *otel.Tracer para instrumentación OpenTelemetry
}

// NewHandler crea una nueva instancia del handler
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// GetSugerencia retorna la sugerencia de tipo/horario basada en la hora actual
// GET /api/v1/inventarios/sugerencia
func (h *Handler) GetSugerencia(w http.ResponseWriter, r *http.Request) {
	// TODO: ctx, span := h.tracer.Start(r.Context(), "inventario.sugerencia.get")
	// TODO: defer span.End()

	sugerencia, err := h.service.Sugerir(r.Context())
	if err != nil {
		// TODO: span.RecordError(err)
		// TODO: span.SetAttributes(attribute.String("resultado", "error"))
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	// TODO: span.SetAttributes(attribute.String("resultado", "success"))
	h.respondJSON(w, http.StatusOK, sugerencia)
}

// PostInventario inicia un nuevo conteo
// POST /api/v1/inventarios
func (h *Handler) PostInventario(w http.ResponseWriter, r *http.Request) {
	var req CreateInventarioReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// TODO: Extraer userID y roleID del JWT
	userID := int64(1)
	roleID := int64(1)

	resp, err := h.service.Iniciar(r.Context(), &req, userID, roleID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.respondJSON(w, http.StatusCreated, resp)
}

// GetInventario obtiene un inventario existente
// GET /api/v1/inventarios/{id}
func (h *Handler) GetInventario(w http.ResponseWriter, r *http.Request) {
	// Extraer ID del path - formato: /api/v1/inventarios/{id}
	idStr := r.PathValue("id")
	if idStr == "" {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id requerido")
		return
	}

	var inventarioID int64
	if _, err := parseID(idStr, &inventarioID); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id inválido")
		return
	}

	// TODO: Extraer userID y roleID del JWT
	userID := int64(1)
	roleID := int64(1)

	resp, err := h.service.Buscar(r.Context(), inventarioID, userID, roleID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// PatchItemValor registra el valor real de un item
// PATCH /api/v1/inventarios/{id}/items/{item_id}
func (h *Handler) PatchItemValor(w http.ResponseWriter, r *http.Request) {
	var req PatchItemValorReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// TODO: Extraer IDs del path
	inventarioID := int64(1)
	itemID := int64(1)

	// TODO: Extraer userID del JWT
	userID := int64(1)

	resp, err := h.service.RegistrarValor(r.Context(), inventarioID, itemID, req.ValorReal, userID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// PostConfirmar marca un conteo como completado
// POST /api/v1/inventarios/{id}/confirmar
func (h *Handler) PostConfirmar(w http.ResponseWriter, r *http.Request) {
	// TODO: Extraer ID del path
	inventarioID := int64(1)

	// TODO: Extraer userID del JWT
	userID := int64(1)

	resp, err := h.service.Confirmar(r.Context(), inventarioID, userID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, resp)
}

// DeleteInventario elimina un conteo en progreso
// DELETE /api/v1/inventarios/{id}
func (h *Handler) DeleteInventario(w http.ResponseWriter, r *http.Request) {
	// TODO: Extraer ID del path
	inventarioID := int64(1)

	// TODO: Extraer userID y roleID del JWT
	userID := int64(1)
	roleID := int64(1)

	err := h.service.Eliminar(r.Context(), inventarioID, userID, roleID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetHistorial obtiene el listado de conteos con filtros y paginación
// GET /api/v1/inventarios
func (h *Handler) GetHistorial(w http.ResponseWriter, r *http.Request) {
	// TODO: Parsear query params para filtros
	filtros := &FiltrosInventario{
		Pagina:    1,
		PorPagina: 50,
	}

	// TODO: Extraer userID y roleID del JWT
	userID := int64(1)
	roleID := int64(1)

	resp, err := h.service.Listar(r.Context(), filtros, userID, roleID)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

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
