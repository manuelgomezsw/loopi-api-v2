package realizar

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ServiceImpl implementa la interfaz Service para registrar valores en conteos
type ServiceImpl struct {
	repo    Repository
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *Metrics
}

// Repository interfaz para acceso a datos específico de realizar
type Repository interface {
	GetInventario(ctx context.Context, inventarioID int64) (*core.Inventario, error)
	GetDetalleItem(ctx context.Context, inventarioID, itemID int64) (*core.DetalleInventario, error)
	UpdateDetalle(ctx context.Context, inventarioID, itemID int64, valorReal float64) (*core.DetalleInventario, error)
	GetItems(ctx context.Context, inventarioID int64) ([]ItemDetalle, error)
	GetItemsDetalle(ctx context.Context, inventarioID int64) ([]core.DetalleInventario, error)
}

// NewService crea una nueva instancia del servicio de realizar
func NewService(repo Repository) *ServiceImpl {
	metrics, _ := NewMetrics()
	if metrics == nil {
		metrics = noopMetrics()
	}
	return &ServiceImpl{
		repo:    repo,
		logger:  slog.Default(),
		tracer:  otel.Tracer("inventario.realizar"),
		metrics: metrics,
	}
}

// RegistrarValor registra el valor real de un item en el conteo
func (s *ServiceImpl) RegistrarValor(r *http.Request, inventarioID, itemID int64, req *RegistrarValorRequest) (*RegistrarValorResponse, error) {
	ctx, span := s.tracer.Start(r.Context(), "inventario.realizar.registrar_valor")
	defer span.End()

	span.SetAttributes(
		attribute.Int64("inventario.id", inventarioID),
		attribute.Int64("item.id", itemID),
		attribute.Float64("valor.real", req.ValorReal),
	)

	// Validar valor >= 0
	if req.ValorReal < 0 {
		span.SetAttributes(attribute.String("resultado", "error_validacion"))
		return nil, ErrValorNegativo
	}

	// Obtener conteo y validar estado
	inventario, err := s.repo.GetInventario(ctx, inventarioID)
	if err != nil {
		span.SetAttributes(attribute.String("resultado", "error_no_encontrado"))
		return nil, ErrConteoNoEncontrado
	}

	if inventario.Estado != core.EstadoEnProgreso {
		span.SetAttributes(attribute.String("resultado", "error_estado"))
		return nil, ErrConteoNoProgreso
	}

	// Obtener item y validar pertenencia
	item, err := s.repo.GetDetalleItem(ctx, inventarioID, itemID)
	if err != nil {
		span.SetAttributes(attribute.String("resultado", "error_item_no_encontrado"))
		return nil, ErrItemNoEncontrado
	}

	// Calcular diferencia
	diferencia := req.ValorReal - item.ValorEsperado
	diferenciaPorcentaje := 0.0
	if item.ValorEsperado != 0 {
		diferenciaPorcentaje = (diferencia / item.ValorEsperado) * 100
	}

	s.logger.InfoContext(ctx, "inventario.realizar: registrando valor",
		"inventario_id", inventarioID,
		"item_id", itemID,
		"valor_real", req.ValorReal,
		"valor_esperado", item.ValorEsperado,
		"diferencia", diferencia)

	// UPDATE en BD
	_, dbSpan := s.tracer.Start(ctx, "inventario.realizar.actualizar_detalle")
	defer dbSpan.End()

	_, err = s.repo.UpdateDetalle(ctx, inventarioID, itemID, req.ValorReal)
	if err != nil {
		span.SetAttributes(attribute.String("resultado", "error_db"))
		return nil, err
	}

	span.SetAttributes(attribute.String("resultado", "success"))

	return &RegistrarValorResponse{
		Success:              true,
		ItemID:               itemID,
		ItemCodigo:           item.Nombre,
		ItemDescripcion:      item.Nombre,
		ValorEsperado:        item.ValorEsperado,
		ValorReal:            req.ValorReal,
		Diferencia:           diferencia,
		DiferenciaPorcentaje: diferenciaPorcentaje,
		Unidad:               fmt.Sprintf("unidad"),
		Timestamp:            time.Now(),
	}, nil
}

// GetDetallesInventario obtiene los detalles del inventario para precarga
func (s *ServiceImpl) GetDetallesInventario(r *http.Request, inventarioID int64) (*GetDetallesResponse, error) {
	ctx, span := s.tracer.Start(r.Context(), "inventario.realizar.obtener_detalles")
	defer span.End()

	span.SetAttributes(attribute.Int64("inventario.id", inventarioID))

	inventario, err := s.repo.GetInventario(ctx, inventarioID)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.GetItems(ctx, inventarioID)
	if err != nil {
		return nil, err
	}

	completados := 0
	for _, item := range items {
		if item.ValorReal != nil {
			completados++
		}
	}

	totalItems := len(items)
	porcentajeProgreso := 0.0
	if totalItems > 0 {
		porcentajeProgreso = float64(completados) / float64(totalItems) * 100
	}

	resp := &GetDetallesResponse{
		InventarioID: inventarioID,
		TiendaID:     inventario.TiendaID,
		Estado:       inventario.Estado,
		Items:        items,
		Resumen: ResumenProgreso{
			TotalItems:        totalItems,
			Completados:       completados,
			Pendientes:        totalItems - completados,
			PorcentajeProgreso: porcentajeProgreso,
		},
	}

	span.SetAttributes(
		attribute.Int64("total_items", int64(totalItems)),
		attribute.Int64("completados", int64(completados)),
	)

	return resp, nil
}
