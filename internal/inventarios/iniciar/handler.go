package iniciar

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Handler maneja los requests HTTP específicos a iniciar conteo
type Handler struct {
	service Service
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *Metrics
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
	metrics, _ := NewMetrics()
	if metrics == nil {
		metrics = noopMetrics()
	}
	return &Handler{
		service: service,
		logger:  slog.Default(),
		tracer:  otel.Tracer(otelScope),
		metrics: metrics,
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
	ctx, span := h.tracer.Start(r.Context(), "inventario.iniciar.crear")
	defer span.End()
	startTime := time.Now()

	var req CreateInventarioReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.ErrorContext(ctx, "inventario.iniciar.post: decode error",
			"error", err.Error())
		h.recordMetricasError(ctx, time.Since(startTime), req.TiendaID, ResultValidationError)
		h.respondError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	claims := r.Context().Value(auth.ContextKeyClaims).(*auth.Claims)
	if claims == nil {
		h.logger.WarnContext(ctx, "inventario.iniciar.post: unauthorized",
			"reason", "no JWT claims found")
		h.recordMetricasError(ctx, time.Since(startTime), req.TiendaID, ResultValidationError)
		h.respondError(w, http.StatusUnauthorized, "unauthorized", "Token JWT inválido")
		return
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		h.logger.ErrorContext(ctx, "inventario.iniciar.post: invalid user id",
			"subject", claims.Subject,
			"error", err.Error())
		h.recordMetricasError(ctx, time.Since(startTime), req.TiendaID, ResultValidationError)
		h.respondError(w, http.StatusBadRequest, "invalid_user", "ID de usuario inválido")
		return
	}

	h.logger.InfoContext(ctx, "inventario.iniciar.post: iniciando",
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
		h.logger.WarnContext(ctx, "inventario.iniciar.post: error",
			"user_id", userID,
			"rol", claims.Rol,
			"error", err.Error(),
			"tienda_id", req.TiendaID)
		resultado := resultFromError(err)
		h.recordMetricasError(ctx, time.Since(startTime), req.TiendaID, resultado)
		h.respondError(w, http.StatusBadRequest, "error", err.Error())
		return
	}

	h.logger.InfoContext(ctx, "inventario.iniciar.post: success",
		"user_id", userID,
		"rol", claims.Rol,
		"inventario_id", resp.ID,
		"tienda_id", resp.TiendaID,
		"items_count", len(resp.Items))
	h.recordMetricasSuccess(ctx, time.Since(startTime), req.TiendaID, string(resp.Tipo), int64(len(resp.Items)))
	h.respondJSON(w, http.StatusCreated, resp)
}

func (h *Handler) recordMetricasSuccess(ctx context.Context, duration time.Duration, tiendaID int64, tipoDeterminado string, itemsCount int64) {
	duracionMs := float64(duration.Milliseconds())
	h.metrics.RecordCrearInventario(ctx, duracionMs, ResultSuccess, tiendaID, itemsCount, tipoDeterminado)
}

func (h *Handler) recordMetricasError(ctx context.Context, duration time.Duration, tiendaID int64, resultado string) {
	duracionMs := float64(duration.Milliseconds())
	h.metrics.RecordCrearInventario(ctx, duracionMs, resultado, tiendaID, 0, "")
}

func resultFromError(err error) string {
	if err == nil {
		return ResultSuccess
	}
	if custErr, ok := err.(*Error); ok {
		switch custErr.Code {
		case "conteo_duplicado":
			return ResultConflict
		case "tienda_no_autorizada", "invalid_tipo", "horario_required", "sin_items_contabilizar":
			return ResultValidationError
		case "error_servidor":
			return ResultServerError
		default:
			return ResultServerError
		}
	}
	return ResultServerError
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
