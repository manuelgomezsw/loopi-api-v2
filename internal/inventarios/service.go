package inventarios

import (
	"context"
	"log/slog"
	"time"
)

// Service define la interfaz de la capa de negocio
type Service interface {
	// Iniciar inicia un nuevo conteo de inventario
	Iniciar(ctx context.Context, req *CreateInventarioReq, userID, roleID int64) (*InventarioResp, error)

	// RegistrarValor registra el valor real de un item en un conteo en progreso
	RegistrarValor(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID int64) (*ItemDetailResp, error)

	// Confirmar marca un conteo como completado
	Confirmar(ctx context.Context, inventarioID int64, userID int64) (*InventarioResp, error)

	// Listar obtiene el historial de conteos con filtros y paginación
	Listar(ctx context.Context, filtros *FiltrosInventario, userID int64, roleID int64) (*HistorialResp, error)

	// Buscar obtiene un inventario con todos sus detalles
	Buscar(ctx context.Context, inventarioID int64, userID int64, roleID int64) (*InventarioResp, error)

	// Modificar permite modificar valores de un conteo completado (solo admin)
	Modificar(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID, roleID int64) (*ItemDetailResp, error)

	// Eliminar elimina un conteo en progreso (solo admin)
	Eliminar(ctx context.Context, inventarioID int64, userID, roleID int64) error

	// Sugerir retorna tipo y horario sugeridos según la hora actual
	Sugerir(ctx context.Context) (*SugerenciaResp, error)

	// ValidarTipo valida que el tipo sea válido
	ValidarTipo(tipo Tipo) error

	// ValidarHorario valida que el horario sea válido o null según el tipo
	ValidarHorario(horario *Horario, tipo Tipo) error
}

// FiltrosInventario contiene los criterios de filtrado para el historial
type FiltrosInventario struct {
	TiendaID    *int64
	Tipo        *Tipo
	Estado      *Estado
	Desde       *time.Time
	Hasta       *time.Time
	Pagina      int
	PorPagina   int
}

// ServiceImpl implementa la interfaz Service
type ServiceImpl struct {
	repo   Repository
	logger *slog.Logger
}

// NewService crea una nueva instancia del servicio
func NewService(repo Repository) Service {
	return &ServiceImpl{
		repo:   repo,
		logger: slog.Default(),
	}
}

func (s *ServiceImpl) Iniciar(ctx context.Context, req *CreateInventarioReq, userID, roleID int64) (*InventarioResp, error) {
	s.logger.InfoContext(ctx, "inventario.iniciar: iniciando",
		"tienda_id", req.TiendaID,
		"tipo", req.Tipo)

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

	// Crear inventario
	now := time.Now()
	inv := &Inventario{
		TiendaID:      req.TiendaID,
		Fecha:         now,
		Tipo:          req.Tipo,
		Horario:       req.Horario,
		Estado:        EstadoEnProgreso,
		ResponsableID: userID,
		IniciadoEn:    now,
		CreadoEn:      now,
		ActualizadoEn: now,
	}

	// Guardar en BD
	createdInv, err := s.repo.CreateInventario(ctx, inv)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error creando",
			"tienda_id", req.TiendaID,
			"error", err.Error())
		return nil, err
	}

	// Obtener detalles con valores sugeridos
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

func (s *ServiceImpl) mapInventarioToResp(inv *Inventario) *InventarioResp {
	items := make([]ItemDetailResp, len(inv.Items))
	for i, detail := range inv.Items {
		items[i] = ItemDetailResp{
			ID:            detail.ID,
			ItemID:        detail.ItemID,
			ValorSugerido: detail.ValorSugerido,
			ValorEsperado: detail.ValorEsperado,
			ValorReal:     detail.ValorReal,
			Diferencia:    detail.Diferencia,
		}
	}

	return &InventarioResp{
		ID:            inv.ID,
		TiendaID:      inv.TiendaID,
		Fecha:         inv.Fecha,
		Tipo:          inv.Tipo,
		Horario:       inv.Horario,
		Estado:        inv.Estado,
		ResponsableID: inv.ResponsableID,
		IniciadoEn:    inv.IniciadoEn,
		CompletadoEn:  inv.CompletadoEn,
		Items:         items,
	}
}

func (s *ServiceImpl) RegistrarValor(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID int64) (*ItemDetailResp, error) {
	s.logger.InfoContext(ctx, "inventario.registrar: iniciando",
		"inventario_id", inventarioID,
		"item_id", itemID,
		"valor_real", valorReal)

	// Obtener inventario para verificar estado y responsable
	inv, err := s.repo.GetInventario(ctx, inventarioID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.registrar: error obteniendo",
			"inventario_id", inventarioID,
			"error", err.Error())
		return nil, err
	}

	if inv == nil {
		s.logger.WarnContext(ctx, "inventario.registrar: not found",
			"inventario_id", inventarioID)
		return nil, NewError("not_found", "inventario no encontrado")
	}

	if inv.Estado != EstadoEnProgreso {
		s.logger.WarnContext(ctx, "inventario.registrar: estado inválido",
			"inventario_id", inventarioID,
			"estado", inv.Estado)
		return nil, NewError("conteo_bloqueado", "solo se pueden registrar valores en conteos en progreso")
	}

	if inv.ResponsableID != userID {
		s.logger.WarnContext(ctx, "inventario.registrar: sin autorización",
			"inventario_id", inventarioID,
			"responsable_id", inv.ResponsableID,
			"user_id", userID)
		return nil, NewError("conteo_bloqueado", "solo el responsable puede registrar valores")
	}

	// Actualizar detalle con valor real
	detail, err := s.repo.UpdateDetalle(ctx, inventarioID, itemID, valorReal)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.registrar: error actualizando",
			"inventario_id", inventarioID,
			"item_id", itemID,
			"error", err.Error())
		return nil, err
	}

	s.logger.InfoContext(ctx, "inventario.registrar: success",
		"inventario_id", inventarioID,
		"item_id", itemID,
		"diferencia", detail.Diferencia)

	return &ItemDetailResp{
		ID:            detail.ID,
		ItemID:        detail.ItemID,
		ValorSugerido: detail.ValorSugerido,
		ValorEsperado: detail.ValorEsperado,
		ValorReal:     detail.ValorReal,
		Diferencia:    detail.Diferencia,
	}, nil
}

func (s *ServiceImpl) Confirmar(ctx context.Context, inventarioID int64, userID int64) (*InventarioResp, error) {
	s.logger.InfoContext(ctx, "inventario.confirmar: iniciando",
		"inventario_id", inventarioID)

	// Obtener inventario
	inv, err := s.repo.GetInventarioDetalle(ctx, inventarioID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.confirmar: error obteniendo",
			"inventario_id", inventarioID,
			"error", err.Error())
		return nil, err
	}

	if inv == nil {
		s.logger.WarnContext(ctx, "inventario.confirmar: not found",
			"inventario_id", inventarioID)
		return nil, NewError("not_found", "inventario no encontrado")
	}

	if inv.Estado != EstadoEnProgreso {
		s.logger.WarnContext(ctx, "inventario.confirmar: ya completado",
			"inventario_id", inventarioID,
			"estado", inv.Estado)
		return nil, NewError("ya_completado", "conteo ya está completado")
	}

	if inv.ResponsableID != userID {
		s.logger.WarnContext(ctx, "inventario.confirmar: sin autorización",
			"inventario_id", inventarioID,
			"responsable_id", inv.ResponsableID,
			"user_id", userID)
		return nil, NewError("conteo_bloqueado", "solo el responsable puede confirmar")
	}

	// Verificar que todos los items tengan valor_real
	var itemsSinRegistrar []int64
	for _, item := range inv.Items {
		if item.ValorReal == nil {
			itemsSinRegistrar = append(itemsSinRegistrar, item.ItemID)
		}
	}

	if len(itemsSinRegistrar) > 0 {
		s.logger.WarnContext(ctx, "inventario.confirmar: items sin registrar",
			"inventario_id", inventarioID,
			"items_count", len(itemsSinRegistrar))
		return nil, NewError("items_sin_registrar", "todos los items deben tener un valor registrado")
	}

	// Confirmar (marcar como completado)
	confirmedInv, err := s.repo.ConfirmarInventario(ctx, inventarioID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.confirmar: error confirmando",
			"inventario_id", inventarioID,
			"error", err.Error())
		return nil, err
	}

	confirmedInv, err = s.repo.GetInventarioDetalle(ctx, confirmedInv.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.confirmar: error obteniendo detalles",
			"inventario_id", inventarioID,
			"error", err.Error())
		return nil, err
	}

	s.logger.InfoContext(ctx, "inventario.confirmar: success",
		"inventario_id", confirmedInv.ID,
		"completado_en", confirmedInv.CompletadoEn,
		"items_count", len(confirmedInv.Items))

	return s.mapInventarioToResp(confirmedInv), nil
}

func (s *ServiceImpl) Listar(ctx context.Context, filtros *FiltrosInventario, userID int64, roleID int64) (*HistorialResp, error) {
	// TODO: Verificar autorización según rol
	// admin: puede listar todas las tiendas
	// lider_tienda: solo su tienda
	// barista: no tiene acceso

	inventarios, total, err := s.repo.ListInventarios(ctx, filtros)
	if err != nil {
		return nil, err
	}

	items := make([]InventarioResp, len(inventarios))
	for i, inv := range inventarios {
		items[i] = *s.mapInventarioToResp(inv)
	}

	totalPages := (total + int64(filtros.PorPagina) - 1) / int64(filtros.PorPagina)

	return &HistorialResp{
		Inventarios:  items,
		Total:        total,
		Pagina:       filtros.Pagina,
		TotalPaginas: int(totalPages),
	}, nil
}

func (s *ServiceImpl) Buscar(ctx context.Context, inventarioID int64, userID int64, roleID int64) (*InventarioResp, error) {
	// TODO: Verificar autorización
	inv, err := s.repo.GetInventarioDetalle(ctx, inventarioID)
	if err != nil {
		return nil, err
	}

	if inv == nil {
		return nil, NewError("not_found", "inventario no encontrado")
	}

	return s.mapInventarioToResp(inv), nil
}

func (s *ServiceImpl) Modificar(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID, roleID int64) (*ItemDetailResp, error) {
	// TODO: Solo admin puede modificar completados
	inv, err := s.repo.GetInventario(ctx, inventarioID)
	if err != nil {
		return nil, err
	}

	if inv == nil {
		return nil, NewError("not_found", "inventario no encontrado")
	}

	if inv.Estado != EstadoCompletado {
		return nil, NewError("estado_invalido", "solo se pueden modificar conteos completados")
	}

	detail, err := s.repo.UpdateDetalleCompletado(ctx, inventarioID, itemID, valorReal)
	if err != nil {
		return nil, err
	}

	return &ItemDetailResp{
		ID:            detail.ID,
		ItemID:        detail.ItemID,
		ValorSugerido: detail.ValorSugerido,
		ValorEsperado: detail.ValorEsperado,
		ValorReal:     detail.ValorReal,
		Diferencia:    detail.Diferencia,
	}, nil
}

func (s *ServiceImpl) Eliminar(ctx context.Context, inventarioID int64, userID, roleID int64) error {
	// TODO: Solo admin puede eliminar en_progreso
	inv, err := s.repo.GetInventario(ctx, inventarioID)
	if err != nil {
		return err
	}

	if inv == nil {
		return NewError("not_found", "inventario no encontrado")
	}

	if inv.Estado != EstadoEnProgreso {
		return NewError("eliminacion_no_permitida", "solo se pueden eliminar conteos en progreso")
	}

	return s.repo.DeleteInventario(ctx, inventarioID)
}

func (s *ServiceImpl) Sugerir(ctx context.Context) (*SugerenciaResp, error) {
	now := time.Now()
	hour := now.Hour()

	// Sugerir tipo y horario según hora del día (Colombia)
	// 06:00-10:59 → diario/apertura
	// 11:00-16:59 → diario/mediodia
	// 17:00-23:59 → diario/cierre
	// 00:00-05:59 → no sugerir diario, permitir otro tipo
	var tipo Tipo
	var horario Horario

	if hour >= 6 && hour < 11 {
		tipo = TipoDiario
		horario = HorarioApertura
	} else if hour >= 11 && hour < 17 {
		tipo = TipoDiario
		horario = HorarioMediodia
	} else if hour >= 17 {
		tipo = TipoDiario
		horario = HoriarioCierre
	} else {
		// Madrugada: sugerir conteo semanal sin horario
		tipo = TipoSemanal
	}

	return &SugerenciaResp{
		Tipo:    tipo,
		Horario: horario,
	}, nil
}

func (s *ServiceImpl) ValidarTipo(tipo Tipo) error {
	switch tipo {
	case TipoDiario, TipoSemanal, TipoMensual, TipoInicial:
		return nil
	default:
		return NewError("invalid_tipo", "tipo debe ser diario, semanal, mensual o inicial")
	}
}

func (s *ServiceImpl) ValidarHorario(horario *Horario, tipo Tipo) error {
	if tipo == TipoDiario {
		if horario == nil {
			return NewError("horario_required", "horario es requerido para conteo diario")
		}
		switch *horario {
		case HorarioApertura, HorarioMediodia, HoriarioCierre:
			return nil
		default:
			return NewError("invalid_horario", "horario debe ser apertura, mediodia o cierre")
		}
	}

	if horario != nil && (tipo == TipoSemanal || tipo == TipoMensual || tipo == TipoInicial) {
		return NewError("horario_not_allowed", "horario no debe ser especificado para conteos semanales, mensuales o iniciales")
	}
	return nil
}

// Error es una estructura de error de negocio
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(code, msg string) *Error {
	return &Error{Code: code, Message: msg}
}
