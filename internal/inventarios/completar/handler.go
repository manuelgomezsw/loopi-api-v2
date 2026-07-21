package completar

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Handler maneja los requests HTTP para confirmar conteos
type Handler struct {
	service Service
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *Metrics
}

// Service interfaz que debe implementar el service de completar
type Service interface {
	ConfirmarConteo(ctx *http.Request, inventarioID, userID int64, userRole string) (*ConfirmarResponse, error)
}

// ErrorResp es la respuesta de error estándar
type ErrorResp struct {
	Error   string      `json:"error"`
	Mensaje string      `json:"mensaje"`
	Campo   *string     `json:"campo,omitempty"`
}

// NewHandler crea una nueva instancia del handler para completar
func NewHandler(service Service) *Handler {
	metrics, _ := NewMetrics()
	if metrics == nil {
		metrics = noopMetrics()
	}
	return &Handler{
		service: service,
		logger:  slog.Default(),
		tracer:  otel.Tracer("inventario.completar"),
		metrics: metrics,
	}
}

// PostConfirmar confirma la finalización de un conteo
// POST /api/v1/inventarios/{id}/confirmar
func (h *Handler) PostConfirmar(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), SpanConfirmar)
	defer span.End()
	startTime := time.Now()

	// Extraer parámetro de ruta
	inventarioIDStr := r.PathValue("id")
	if inventarioIDStr == "" {
		h.logger.ErrorContext(ctx, "inventario.completar.post: inventario id missing")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id requerido")
		return
	}

	var inventarioID int64
	if _, err := strconv.ParseInt(inventarioIDStr, 10, 64); err != nil {
		inventarioID, _ = strconv.ParseInt(inventarioIDStr, 10, 64)
		if err != nil {
			h.logger.ErrorContext(ctx, "inventario.completar.post: invalid inventario id",
				"id_str", inventarioIDStr,
				"error", err.Error())
			h.respondError(w, http.StatusBadRequest, "invalid_request", "inventario id inválido")
			return
		}
	}

	// Verificar autenticación
	claims := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.completar.post: unauthorized")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.completar.post: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	// Decodificar request body
	var req ConfirmarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(ctx, "inventario.completar.post: decode error", "error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if !req.Confirmar {
		h.logger.WarnContext(ctx, "inventario.completar.post: confirmar = false")
		h.respondError(w, http.StatusBadRequest, "invalid_request", "confirmar debe ser true")
		return
	}

	h.logger.InfoContext(ctx, "inventario.completar.post: iniciando",
		"user_id", userID,
		"role", claims.Rol,
		"inventario_id", inventarioID)

	// Llamar al service
	resp, err := h.service.ConfirmarConteo(r, inventarioID, userID, claims.Rol)
	if err != nil {
		h.logger.WarnContext(ctx, "inventario.completar.post: error del service", "error", err.Error())
		// Mapear error a código HTTP apropiado
		statusCode := http.StatusInternalServerError
		errorCode := "internal_error"
		errorMsg := err.Error()

		if custErr, ok := err.(*Error); ok {
			errorCode = custErr.Code
			errorMsg = custErr.Message
			switch custErr.Code {
			case "CONTEO_INCOMPLETO":
				statusCode = http.StatusUnprocessableEntity // 422
			case "PERMISO_INSUFICIENTE":
				statusCode = http.StatusForbidden // 403
			case "CONTEO_NO_ENCONTRADO":
				statusCode = http.StatusNotFound // 404
			case "CONTEO_YA_COMPLETADO", "CONTEO_NO_EN_PROGRESO":
				statusCode = http.StatusConflict // 409
			}
		}
		h.respondError(w, statusCode, errorCode, errorMsg)
		return
	}

	h.logger.InfoContext(ctx, "inventario.completar.post: success",
		"inventario_id", resp.ID,
		"items_ajustados", resp.ItemsAjustados)

	// Registrar métricas
	duracionMs := float64(time.Since(startTime).Milliseconds())
	var tiendaID *int64
	if claims.TiendaID != nil {
		tid := int64(*claims.TiendaID)
		tiendaID = &tid
	}
	h.metrics.RecordConfirmar(ctx, tiendaID, duracionMs, "success")

	h.respondJSON(w, http.StatusOK, resp)
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
