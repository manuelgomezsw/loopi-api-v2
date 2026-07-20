package iniciar

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// ServiceImpl implementa la interfaz Service para iniciar conteos
type ServiceImpl struct {
	repo    Repository
	logger  *slog.Logger
	tracer  trace.Tracer
	metrics *Metrics
}

// Repository interfaz para acceso a datos específico de iniciar
type Repository interface {
	CreateInventario(ctx context.Context, inv *core.Inventario) (*core.Inventario, error)
	CreateDetalleInventario(ctx context.Context, detalles []core.DetalleInventario) error
	GetItemsActivosPorTipo(ctx context.Context, tiendaID int64, tipo core.Tipo) ([]int64, error)
	GetStockSnapshot(ctx context.Context, tiendaID int64, itemIDs []int64) (map[int64]float64, error)
	ExisteInventarioCompletado(ctx context.Context, tiendaID int64) (bool, error)
	GetInventarioDetalle(ctx context.Context, inventarioID int64) (*core.Inventario, error)
}

// NewService crea una nueva instancia del servicio de iniciar
func NewService(repo Repository) Service {
	metrics, _ := NewMetrics()
	if metrics == nil {
		metrics = noopMetrics()
	}
	return &ServiceImpl{
		repo:    repo,
		logger:  slog.Default(),
		tracer:  otel.Tracer(otelScope),
		metrics: metrics,
	}
}


// Sugerir retorna la sugerencia de tipo/horario basada en la hora actual
func (s *ServiceImpl) Sugerir(r *http.Request) (*SugerenciaResp, error) {
	now := time.Now()
	hour := now.Hour()

	tipo := core.TipoDiario
	var horario core.Horario

	if hour >= 6 && hour < 11 {
		horario = core.HorarioApertura
	} else if hour >= 11 && hour < 15 {
		horario = core.HorarioMediodia
	} else if hour >= 15 {
		horario = core.HoriarioCierre
	} else {
		horario = core.HoriarioCierre
	}

	return &SugerenciaResp{
		Tipo:    tipo,
		Horario: horario,
	}, nil
}

// Iniciar inicia un nuevo conteo de inventario
func (s *ServiceImpl) Iniciar(r *http.Request, req *CreateInventarioReq, userID int64, rol string, tiendaID *int64) (*InventarioResp, error) {
	ctx := r.Context()

	s.logger.InfoContext(ctx, "inventario.iniciar: iniciando",
		"user_id", userID,
		"user_role", rol,
		"tienda_id", req.TiendaID,
		"tipo", req.Tipo)

	// RBAC: Validar autorización por tienda (RF-INV-01.1)
	if rol != "admin" {
		if tiendaID == nil || *tiendaID != req.TiendaID {
			s.logger.WarnContext(ctx, "inventario.iniciar: unauthorized tienda",
				"user_id", userID,
				"user_role", rol,
				"user_tienda_id", tiendaID,
				"requested_tienda_id", req.TiendaID)
			return nil, NewError("tienda_no_autorizada", "No tienes permiso para iniciar conteos en esta tienda")
		}
	}

	// Validar tipo
	if err := s.ValidarTipo(req.Tipo); err != nil {
		s.logger.WarnContext(ctx, "inventario.iniciar: tipo inválido",
			"tipo", req.Tipo,
			"error", err.Error())
		return nil, err
	}

	// Validar horario según tipo
	if err := s.ValidarHorario(req.Horario, req.Tipo); err != nil {
		s.logger.WarnContext(ctx, "inventario.iniciar: horario inválido",
			"tipo", req.Tipo,
			"error", err.Error())
		return nil, err
	}

	// Obtener items activos para el tipo (con span de observabilidad)
	ctxCargar, spanCargar := s.tracer.Start(ctx, "inventario.iniciar.cargar_items")
	startCargar := time.Now()
	itemIDs, err := s.repo.GetItemsActivosPorTipo(ctxCargar, req.TiendaID, req.Tipo)
	duracionCargar := time.Since(startCargar).Milliseconds()
	s.metrics.RecordCargarItems(ctxCargar, float64(duracionCargar))
	spanCargar.End()
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error obteniendo items",
			"tienda_id", req.TiendaID,
			"tipo", req.Tipo,
			"error", err.Error())
		return nil, NewError("error_servidor", "No se pudo obtener la lista de items para contar.")
	}

	// Validar hay items para contar (RF-INV-02.3)
	if len(itemIDs) == 0 {
		s.logger.WarnContext(ctx, "inventario.iniciar: sin items para tipo",
			"tienda_id", req.TiendaID,
			"tipo", req.Tipo)
		return nil, NewError("sin_items_contabilizar",
			fmt.Sprintf("No hay items para contar de tipo %v.", req.Tipo))
	}

	// Obtener snapshot de stock
	stockSnapshot, err := s.repo.GetStockSnapshot(ctx, req.TiendaID, itemIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error obteniendo snapshot",
			"tienda_id", req.TiendaID,
			"error", err.Error())
		return nil, NewError("error_servidor", "No se pudo preparar los datos de stock.")
	}

	// Determinación automática de tipo según historial (con span de observabilidad)
	ctxDeterminar, spanDeterminar := s.tracer.Start(ctx, "inventario.iniciar.determinar_tipo")
	startDeterminar := time.Now()
	existeHistorial, err := s.repo.ExisteInventarioCompletado(ctxDeterminar, req.TiendaID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error consultando historial",
			"tienda_id", req.TiendaID,
			"error", err.Error())
	}

	// Determinar tipo_real según historial
	tipoReal := req.Tipo
	if !existeHistorial {
		// Primer conteo: determinar como "inicial"
		tipoReal = core.TipoInicial
		s.logger.InfoContext(ctx, "inventario.iniciar: determinado como inicial (primer conteo)",
			"tienda_id", req.TiendaID,
			"tipo_solicitado", req.Tipo)
	}
	duracionDeterminar := time.Since(startDeterminar).Milliseconds()
	s.metrics.RecordDeterminarTipo(ctxDeterminar, float64(duracionDeterminar))
	spanDeterminar.End()

	// Si tipo es inicial, anular horario
	if tipoReal == core.TipoInicial {
		req.Horario = nil
	}

	// Crear inventario
	now := time.Now()
	inv := &core.Inventario{
		TiendaID:      req.TiendaID,
		Fecha:         now,
		Tipo:          tipoReal,
		Horario:       req.Horario,
		Estado:        core.EstadoEnProgreso,
		ResponsableID: userID,
		IniciadoEn:    now,
		CreadoEn:      now,
		ActualizadoEn: now,
	}

	createdInv, err := s.repo.CreateInventario(ctx, inv)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error creando",
			"tienda_id", req.TiendaID,
			"error", err.Error())
		return nil, err
	}

	// Crear detalles con valor_esperado desde snapshot
	detalles := make([]core.DetalleInventario, len(itemIDs))
	for i, itemID := range itemIDs {
		detalles[i] = core.DetalleInventario{
			InventarioID:  createdInv.ID,
			ItemID:        itemID,
			ValorEsperado: stockSnapshot[itemID],
			CreadoEn:      now,
			ActualizadoEn: now,
		}
	}

	err = s.repo.CreateDetalleInventario(ctx, detalles)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error creando detalles",
			"inventario_id", createdInv.ID,
			"items_count", len(detalles),
			"error", err.Error())
		return nil, err
	}

	// Obtener inventario completo con detalles
	createdInv, err = s.repo.GetInventarioDetalle(ctx, createdInv.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error obteniendo detalles",
			"inventario_id", createdInv.ID,
			"error", err.Error())
		return nil, err
	}

	s.logger.InfoContext(ctx, "inventario.iniciar: success",
		"inventario_id", createdInv.ID,
		"tienda_id", createdInv.TiendaID,
		"items_count", len(createdInv.Items))

	return s.mapInventarioToResp(createdInv), nil
}

// ValidarTipo valida que el tipo sea válido
func (s *ServiceImpl) ValidarTipo(tipo core.Tipo) error {
	if tipo == core.TipoInicial {
		return NewError("tipo_inicial_no_permitido",
			"El tipo 'inicial' no puede ser seleccionado manualmente.")
	}

	switch tipo {
	case core.TipoDiario, core.TipoSemanal, core.TipoMensual:
		return nil
	default:
		return NewError("invalid_tipo", "tipo debe ser diario, semanal o mensual")
	}
}

// ValidarHorario valida que el horario sea válido según el tipo
func (s *ServiceImpl) ValidarHorario(horario *core.Horario, tipo core.Tipo) error {
	if tipo == core.TipoDiario {
		if horario == nil {
			return NewError("horario_required", "horario es requerido para conteo diario")
		}
		switch *horario {
		case core.HorarioApertura, core.HorarioMediodia, core.HoriarioCierre:
			return nil
		default:
			return NewError("invalid_horario", "horario debe ser apertura, mediodia o cierre")
		}
	}

	if horario != nil && (tipo == core.TipoSemanal || tipo == core.TipoMensual || tipo == core.TipoInicial) {
		return NewError("horario_not_allowed", "horario no debe ser especificado para conteos semanales o mensuales")
	}
	return nil
}

func (s *ServiceImpl) mapInventarioToResp(inv *core.Inventario) *InventarioResp {
	items := make([]ItemDetailResp, len(inv.Items))
	for i, detail := range inv.Items {
		items[i] = ItemDetailResp{
			ID:            detail.ID,
			ItemID:        detail.ItemID,
			Nombre:        detail.Nombre,
			ValorEsperado: detail.ValorEsperado,
			ValorReal:     detail.ValorReal,
		}
	}

	return &InventarioResp{
		ID:            inv.ID,
		TiendaID:      inv.TiendaID,
		Tipo:          inv.Tipo,
		Horario:       inv.Horario,
		Estado:        inv.Estado,
		ResponsableID: inv.ResponsableID,
		IniciadoEn:    inv.IniciadoEn.Format(time.RFC3339),
		Items:         items,
	}
}
