package iniciar

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"go.opentelemetry.io/otel"
)

// Handler maneja los requests HTTP específicos a iniciar conteo
type Handler struct {
	service Service
	logger  *slog.Logger
	tracer  interface{}
}

// Service interfaz que debe implementar el service de iniciar
type Service interface {
	Sugerir(ctx *http.Request) (*SugerenciaResp, error)
	Iniciar(ctx *http.Request, req *CreateInventarioReq, userID int64, rol string, tiendaID *int64) (*InventarioResp, error)
}


// ErrorResp es la respuesta de error estándar
type ErrorResp struct {
	Error    string      `json:"error"`
	Mensaje  string      `json:"mensaje"`
	Campo    *string     `json:"campo,omitempty"`
	Detalles interface{} `json:"detalles,omitempty"`
}

// NewHandler crea una nueva instancia del handler para iniciar
func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
		logger:  slog.Default(),
		tracer:  otel.Tracer("inventarios.iniciar"),
	}
}

// GetSugerencia retorna la sugerencia de tipo/horario basada en la hora actual
// GET /api/v1/inventarios/sugerencia
func (h *Handler) GetSugerencia(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	userID, _ := strconv.ParseInt(claims.Subject, 10, 64)

	h.logger.InfoContext(r.Context(), "inventario.iniciar.sugerencia: iniciando",
		"user_id", userID,
		"rol", claims.Rol)

	sugerencia, err := h.service.Sugerir(r)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "inventario.iniciar.sugerencia: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(r.Context(), "inventario.iniciar.sugerencia: success",
		"user_id", userID,
		"rol", claims.Rol,
		"tipo", sugerencia.Tipo)
	h.respondJSON(w, http.StatusOK, sugerencia)
}

// PostInventario inicia un nuevo conteo
// POST /api/v1/inventarios
func (h *Handler) PostInventario(w http.ResponseWriter, r *http.Request) {
	var req CreateInventarioReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(r.Context(), "inventario.iniciar.post: decode error",
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	claims := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(r.Context(), "inventario.iniciar.post: unauthorized",
			"reason", "no JWT claims found")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "inventario.iniciar.post: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(r.Context(), "inventario.iniciar.post: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"tienda_id", req.TiendaID,
		"tipo", req.Tipo)

	var tiendaID *int64
	if claims.TiendaID != nil {
		tid := int64(*claims.TiendaID)
		tiendaID = &tid
	}

	resp, err := h.service.Iniciar(r, &req, userID, claims.Rol, tiendaID)
	if err != nil {
		h.logger.WarnContext(r.Context(), "inventario.iniciar.post: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error", err.Error(),
			"tienda_id", req.TiendaID)
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(r.Context(), "inventario.iniciar.post: success",
		"user_id", userID,
		"rol", claims.Rol,
		"inventario_id", resp.ID,
		"tienda_id", resp.TiendaID)
	h.respondJSON(w, http.StatusCreated, resp)
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) respondError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResp{
		Error:   errCode,
		Mensaje: message,
	})
}
