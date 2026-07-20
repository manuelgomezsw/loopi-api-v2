package completar

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ServiceImpl implementa la interfaz Service para confirmar conteos
type ServiceImpl struct {
	repo    Repository
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *Metrics
}

// Repository interfaz para acceso a datos específico de completar
type Repository interface {
	GetInventarioDetalle(ctx context.Context, inventarioID int64) (*core.Inventario, error)
	ConfirmarInventario(ctx context.Context, tx *sql.Tx, inventarioID int64) error
	UpdateStockActual(ctx context.Context, tx *sql.Tx, tiendaID, itemID int64, nuevoValor float64) error
	RecordDifference(ctx context.Context, tx *sql.Tx, detalleID int64, diferencia float64) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CommitTx(tx *sql.Tx) error
	RollbackTx(tx *sql.Tx) error
}

// NewService crea una nueva instancia del servicio de completar
func NewService(repo Repository) *ServiceImpl {
	metrics, _ := NewMetrics()
	if metrics == nil {
		metrics = noopMetrics()
	}
	return &ServiceImpl{
		repo:    repo,
		logger:  slog.Default(),
		tracer:  otel.Tracer("inventario.completar"),
		metrics: metrics,
	}
}

// ConfirmarConteo confirma la finalización de un conteo y ajusta stock
func (s *ServiceImpl) ConfirmarConteo(r *http.Request, inventarioID, userID int64, userRole string) (*ConfirmarResponse, error) {
	ctx, span := s.tracer.Start(r.Context(), SpanConfirmar)
	defer span.End()
	startTime := time.Now()

	span.SetAttributes(
		attribute.Int64("inventario.id", inventarioID),
		attribute.Int64("user.id", userID),
		attribute.String("user.role", userRole),
	)

	s.logger.InfoContext(ctx, "inventario.completar: iniciando confirmación",
		"inventario_id", inventarioID,
		"user_id", userID,
		"user_role", userRole)

	// 1. Obtener inventario
	inventario, err := s.repo.GetInventarioDetalle(ctx, inventarioID)
	if err != nil {
		span.SetAttributes(attribute.String("resultado", "error_no_encontrado"))
		s.logger.ErrorContext(ctx, "inventario.completar: error obteniendo inventario", "error", err)
		return nil, ErrConteoNoEncontrado
	}

	if inventario == nil {
		span.SetAttributes(attribute.String("resultado", "error_no_encontrado"))
		return nil, ErrConteoNoEncontrado
	}

	// 2. Validar estado
	if inventario.Estado != core.EstadoEnProgreso {
		span.SetAttributes(attribute.String("resultado", "error_ya_completado"))
		return nil, ErrConteoNoEnProgreso
	}

	// 3. Validar permisos (responsable o admin)
	if userRole != "admin" && inventario.ResponsableID != userID {
		span.SetAttributes(attribute.String("resultado", "error_permiso"))
		return nil, ErrPermisoInsuficiente
	}

	// 4. Validar completitud: todos los items con valor_real
	validarCtx, validarSpan := s.tracer.Start(ctx, SpanValidarCompletitud)
	itemsIncompletos := 0
	for _, item := range inventario.Items {
		if item.ValorReal == nil {
			itemsIncompletos++
		}
	}
	validarSpan.SetAttributes(attribute.Int("items.incompletos", itemsIncompletos))
	validarSpan.End()

	if itemsIncompletos > 0 {
		span.SetAttributes(attribute.String("resultado", "error_incompleto"))
		s.logger.WarnContext(ctx, "inventario.completar: items incompletos", "count", itemsIncompletos)
		return nil, ErrConteoIncompleto
	}

	// 5. Iniciar transacción
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		span.SetAttributes(attribute.String("resultado", "error_transaccion"))
		return nil, ErrTransaccionFallo
	}
	defer func() {
		if err := recover(); err != nil {
			_ = s.repo.RollbackTx(tx)
		}
	}()

	// 6. Actualizar stock y registrar diferencias
	actualizarCtx, actualizarSpan := s.tracer.Start(ctx, SpanActualizarStock)
	itemsAjustados := 0
	var resumen ResumenConfirmacion
	resumen.TotalItems = len(inventario.Items)

	for _, item := range inventario.Items {
		if item.ValorReal != nil {
			// Calcular diferencia
			diferencia := *item.ValorReal - item.ValorEsperado

			// Ajustar stock
			if err := s.repo.UpdateStockActual(actualizarCtx, tx, inventario.TiendaID, item.ItemID, *item.ValorReal); err != nil {
				_ = s.repo.RollbackTx(tx)
				actualizarSpan.End()
				return nil, err
			}

			// Registrar diferencia
			if err := s.repo.RecordDifference(actualizarCtx, tx, item.ID, diferencia); err != nil {
				_ = s.repo.RollbackTx(tx)
				actualizarSpan.End()
				return nil, err
			}

			itemsAjustados++

			// Contar por tipo de diferencia
			if diferencia == 0 {
				resumen.ItemsCorrectos++
			} else if diferencia < 0 {
				resumen.ItemsFaltantes++
				resumen.DiferenciaNegativa += -diferencia
			} else {
				resumen.ItemsExceso++
				resumen.DiferenciaPositiva += diferencia
			}
		}
	}

	// Calcular porcentaje de variación
	if resumen.TotalItems > 0 {
		resumen.PorcentajeVariacion = ((resumen.DiferenciaPositiva - resumen.DiferenciaNegativa) / (resumen.DiferenciaNegativa + resumen.DiferenciaPositiva)) * 100
	}

	actualizarSpan.SetAttributes(
		attribute.Int("items.ajustados", itemsAjustados),
		attribute.Int("items.correctos", resumen.ItemsCorrectos),
		attribute.Int("items.faltantes", resumen.ItemsFaltantes),
		attribute.Int("items.exceso", resumen.ItemsExceso),
	)
	actualizarSpan.End()

	// 7. Confirmar inventario (cambiar estado a completado)
	if err := s.repo.ConfirmarInventario(ctx, tx, inventarioID); err != nil {
		_ = s.repo.RollbackTx(tx)
		span.SetAttributes(attribute.String("resultado", "error_confirmar"))
		return nil, err
	}

	// 8. Commit
	if err := s.repo.CommitTx(tx); err != nil {
		span.SetAttributes(attribute.String("resultado", "error_commit"))
		return nil, ErrTransaccionFallo
	}

	span.SetAttributes(
		attribute.String("resultado", "success"),
		attribute.Int("items.ajustados", itemsAjustados),
	)

	s.logger.InfoContext(ctx, "inventario.completar: success",
		"inventario_id", inventarioID,
		"items_ajustados", itemsAjustados)

	// 9. Registrar métricas
	duracionMs := float64(time.Since(startTime).Milliseconds())
	s.metrics.RecordConfirmar(ctx, &inventario.TiendaID, duracionMs, "success")
	s.metrics.RecordItemsAjustados(ctx, int64(itemsAjustados), &inventario.TiendaID)

	// 10. Preparar respuesta
	completadoEn := time.Now()
	return &ConfirmarResponse{
		ID:             inventario.ID,
		TiendaID:       inventario.TiendaID,
		Estado:         core.EstadoCompletado,
		CompletadoEn:   completadoEn,
		ItemsAjustados: itemsAjustados,
		Resumen:        &resumen,
		Timestamp:      completadoEn,
	}, nil
}
