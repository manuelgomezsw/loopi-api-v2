package realizar

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

// Handler maneja los requests HTTP para registrar valores en conteos
type Handler struct {
	service Service
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *Metrics
}

// Service interfaz que debe implementar el service de realizar
type Service interface {
	RegistrarValor(r *http.Request, inventarioID, itemID int64, req *RegistrarValorRequest) (*RegistrarValorResponse, error)
	GetDetallesInventario(r *http.Request, inventarioID int64) (*GetDetallesResponse, error)
}

// ErrorResp es la respuesta de error estándar
type ErrorResp struct {
	Error   string      `json:"error"`
	Mensaje string      `json:"mensaje"`
	Campo   *string     `json:"campo,omitempty"`
}

// NewHandler crea una nueva instancia del handler para realizar
func NewHandler(service Service) *Handler {
	metrics, _ := NewMetrics()
	if metrics == nil {
		metrics = noopMetrics()
	}
	return &Handler{
		service: service,
		logger:  slog.Default(),
		tracer:  otel.Tracer("inventario.realizar"),
		metrics: metrics,
	}
}

// PostRegistrarValor registra un valor en un item del conteo
// POST /api/v1/inventarios/{inventario_id}/items/{item_id}/valor
func (h *Handler) PostRegistrarValor(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "inventario.realizar.handler.registrar")
	defer span.End()
	startTime := time.Now()

	// Extraer parámetros de ruta
	inventarioID := r.PathValue("inventario_id")
	itemID := r.PathValue("item_id")

	// Validar parámetros no vacíos
	if inventarioID == "" || itemID == "" {
		h.logger.ErrorContext(ctx, "inventario.realizar.post: parámetros vacíos")
		h.respondError(w, http.StatusBadRequest, "invalid_params", "inventario_id e item_id son requeridos")
		return
	}

	// Convertir a int64
	invID, err := strconv.ParseInt(inventarioID, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.realizar.post: inventario_id inválido", "value", inventarioID)
		h.respondError(w, http.StatusBadRequest, "invalid_inventario_id", "inventario_id debe ser un número")
		return
	}

	itemIDInt, err := strconv.ParseInt(itemID, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.realizar.post: item_id inválido", "value", itemID)
		h.respondError(w, http.StatusBadRequest, "invalid_item_id", "item_id debe ser un número")
		return
	}

	// Decodificar request body
	var req RegistrarValorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(ctx, "inventario.realizar.post: decode error", "error", err.Error())
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Validar valor >= 0
	if req.ValorReal < 0 {
		h.logger.WarnContext(ctx, "inventario.realizar.post: valor negativo", "valor", req.ValorReal)
		h.respondError(w, http.StatusBadRequest, "invalid_valor", "valor_real debe ser >= 0")
		return
	}

	// Verificar autenticación
	claims := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.realizar.post: unauthorized")
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.realizar.post: iniciando",
		"inventario_id", invID,
		"item_id", itemIDInt,
		"valor_real", req.ValorReal)

	// Llamar al service
	resp, err := h.service.RegistrarValor(r, invID, itemIDInt, &req)
	if err != nil {
		h.logger.WarnContext(ctx, "inventario.realizar.post: error del service", "error", err.Error())
		// Mapear error a código HTTP apropiado
		statusCode := http.StatusInternalServerError
		if custErr, ok := err.(*Error); ok {
			switch custErr.Code {
			case "CONTEO_NO_EN_PROGRESO":
				statusCode = http.StatusConflict
			case "ITEM_NO_ENCONTRADO":
				statusCode = http.StatusNotFound
			case "INVALID_VALOR":
				statusCode = http.StatusBadRequest
			}
		}
		h.respondError(w, statusCode, err.Error(), err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.realizar.post: success",
		"inventario_id", invID,
		"item_id", itemIDInt,
		"valor_real", resp.ValorReal,
		"diferencia", resp.Diferencia)

	// Registrar métricas
	duracionMs := float64(time.Since(startTime).Milliseconds())
	var tiendaID *int64
	if claims.TiendaID != nil {
		tid := int64(*claims.TiendaID)
		tiendaID = &tid
	}
	h.metrics.RecordRegistrar(ctx, tiendaID, duracionMs, "success")

	h.respondJSON(w, http.StatusOK, resp)
}

// GetPrecarga obtiene los detalles del inventario para precarga
// GET /api/v1/inventarios/{inventario_id}/detalles
func (h *Handler) GetPrecarga(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "inventario.realizar.handler.precarga")
	defer span.End()

	inventarioID := r.PathValue("inventario_id")
	if inventarioID == "" {
		h.logger.ErrorContext(ctx, "inventario.realizar.get: inventario_id vacío")
		h.respondError(w, http.StatusBadRequest, "invalid_params", "inventario_id es requerido")
		return
	}

	invID, err := strconv.ParseInt(inventarioID, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.realizar.get: inventario_id inválido", "value", inventarioID)
		h.respondError(w, http.StatusBadRequest, "invalid_inventario_id", "inventario_id debe ser un número")
		return
	}

	resp, err := h.service.GetDetallesInventario(r, invID)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.realizar.get: error", "error", err.Error())
		h.respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

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
