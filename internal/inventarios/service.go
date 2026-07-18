package inventarios

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Service define la interfaz de la capa de negocio
type Service interface {
	// Iniciar inicia un nuevo conteo de inventario
	Iniciar(ctx context.Context, req *CreateInventarioReq, userID int64, role string, userTiendaID *int64) (*InventarioResp, error)

	// RegistrarValor registra el valor real de un item en un conteo en progreso
	RegistrarValor(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID int64) (*ItemDetailResp, error)

	// Confirmar marca un conteo como completado
	Confirmar(ctx context.Context, inventarioID int64, userID int64) (*InventarioResp, error)

	// Listar obtiene el historial de conteos con filtros y paginación
	Listar(ctx context.Context, filtros *FiltrosInventario, userID int64, role string, userTiendaID *int64) (*HistorialResp, error)

	// Buscar obtiene un inventario con todos sus detalles
	Buscar(ctx context.Context, inventarioID int64, userID int64, role string, userTiendaID *int64) (*InventarioResp, error)

	// Modificar permite modificar valores de un conteo completado (solo admin)
	Modificar(ctx context.Context, inventarioID, itemID int64, valorReal float64, userID, roleID int64) (*ItemDetailResp, error)

	// Eliminar elimina un conteo en progreso (solo admin)
	Eliminar(ctx context.Context, inventarioID int64, userID int64, role string, userTiendaID *int64) error

	// Sugerir retorna tipo y horario sugeridos según la hora actual
	Sugerir(ctx context.Context) (*SugerenciaResp, error)

	// ValidarTipo valida que el tipo sea válido
	ValidarTipo(tipo Tipo) error

	// ValidarHorario valida que el horario sea válido o null según el tipo
	ValidarHorario(horario *Horario, tipo Tipo) error

	// GetEstadoInventarioActivo verifica si hay un conteo activo en una tienda (T143)
	// Retorna el conteo activo si existe, nil si no hay
	GetEstadoInventarioActivo(ctx context.Context, tiendaID int64) (*InventarioResp, error)
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

func (s *ServiceImpl) Iniciar(ctx context.Context, req *CreateInventarioReq, userID int64, role string, userTiendaID *int64) (*InventarioResp, error) {
	s.logger.InfoContext(ctx, "inventario.iniciar: iniciando",
		"user_id", userID,
		"user_role", role,
		"tienda_id", req.TiendaID,
		"tipo", req.Tipo)

	// RBAC: Validar autorización por tienda (RF-INV-01.1)
	if role != "admin" {
		if userTiendaID == nil || *userTiendaID != req.TiendaID {
			s.logger.WarnContext(ctx, "inventario.iniciar: unauthorized tienda",
				"user_id", userID,
				"user_role", role,
				"user_tienda_id", userTiendaID,
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

	// T156: Paso 1 — Query items ANTES de crear inventario
	// Query: SELECT id FROM items WHERE tienda_id=? AND activo=1 AND frecuencia_inventario=?
	itemIDs, err := s.repo.GetItemsActivosPorTipo(ctx, req.TiendaID, req.Tipo)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error obteniendo items",
			"tienda_id", req.TiendaID,
			"tipo", req.Tipo,
			"error", err.Error())
		return nil, NewError("error_servidor", "No se pudo obtener la lista de items para contar. Por favor intenta de nuevo o contacta al administrador si el problema persiste.")
	}

	// Validar hay items para contar (RF-INV-02.3)
	if len(itemIDs) == 0 {
		s.logger.WarnContext(ctx, "inventario.iniciar: sin items para tipo",
			"tienda_id", req.TiendaID,
			"tipo", req.Tipo)
		return nil, NewError("sin_items_contabilizar", fmt.Sprintf("No hay items para contar de tipo %v. Verifica que haya items activos con esa frecuencia de inventario.", req.Tipo))
	}

	// T157: Paso 2 — Cruzar con stock_actual para obtener valor_sugerido
	// Query: SELECT item_id, valor_snapshot FROM stock_actual WHERE tienda_id=? AND item_id IN (...)
	stockSnapshot, err := s.repo.GetStockSnapshot(ctx, req.TiendaID, itemIDs)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error obteniendo snapshot",
			"tienda_id", req.TiendaID,
			"error", err.Error())
		return nil, NewError("error_servidor", "No se pudo preparar los datos de stock. Por favor intenta de nuevo.")
	}

	// T158: Paso 3 — Crear inventario + detalles (AHORA, después de validaciones)
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

	// Crear inventario
	createdInv, err := s.repo.CreateInventario(ctx, inv)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.iniciar: error creando",
			"tienda_id", req.TiendaID,
			"error", err.Error())
		return nil, err
	}

	// Crear detalles con valor_sugerido mapeado desde stockSnapshot
	detalles := make([]DetalleInventario, len(itemIDs))
	for i, itemID := range itemIDs {
		detalles[i] = DetalleInventario{
			InventarioID:   createdInv.ID,
			ItemID:         itemID,
			ValorSugerido:  stockSnapshot[itemID],
			ValorEsperado:  0,
			CreadoEn:       now,
			ActualizadoEn:  now,
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

	// Obtener inventario completo con detalles creados
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

func (s *ServiceImpl) Listar(ctx context.Context, filtros *FiltrosInventario, userID int64, role string, userTiendaID *int64) (*HistorialResp, error) {
	s.logger.InfoContext(ctx, "inventario.listar: iniciando",
		"user_id", userID,
		"user_role", role,
		"user_tienda_id", userTiendaID)

	// RBAC: Validar autorización según rol (RF-INV-04.1, RF-INV-04.2)
	if role == "barista" {
		s.logger.WarnContext(ctx, "inventario.listar: unauthorized role",
			"user_id", userID,
			"user_role", role)
		return nil, NewError("sin_permiso", "Tu rol no tiene permiso para listar inventarios")
	}

	if role == "lider_tienda" {
		if userTiendaID == nil {
			s.logger.WarnContext(ctx, "inventario.listar: lider_tienda sin tienda",
				"user_id", userID)
			return nil, NewError("sin_tienda", "Tu usuario no tiene una tienda asignada")
		}
		// Filtrar solo por tienda del usuario
		filtros.TiendaID = userTiendaID
	}
	// admin puede listar todas las tiendas (sin restricción)

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

func (s *ServiceImpl) Buscar(ctx context.Context, inventarioID int64, userID int64, role string, userTiendaID *int64) (*InventarioResp, error) {
	inv, err := s.repo.GetInventarioDetalle(ctx, inventarioID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.buscar: error obteniendo",
			"inventario_id", inventarioID,
			"error", err.Error())
		return nil, err
	}

	if inv == nil {
		s.logger.WarnContext(ctx, "inventario.buscar: not found",
			"inventario_id", inventarioID)
		return nil, NewError("not_found", "inventario no encontrado")
	}

	// RBAC: Validar autorización por tienda (RF-INV-04)
	if role != "admin" {
		if userTiendaID == nil || *userTiendaID != inv.TiendaID {
			s.logger.WarnContext(ctx, "inventario.buscar: unauthorized tienda",
				"user_id", userID,
				"user_role", role,
				"user_tienda_id", userTiendaID,
				"inventario_tienda_id", inv.TiendaID)
			return nil, NewError("tienda_no_autorizada", "No tienes permiso para ver inventarios de esta tienda")
		}
	}

	// Validar que user es el responsable (solo no-admin necesita validar)
	if role != "admin" && userID != inv.ResponsableID {
		s.logger.WarnContext(ctx, "inventario.buscar: unauthorized responsable",
			"user_id", userID,
			"user_role", role,
			"inventario_responsable_id", inv.ResponsableID,
			"inventario_id", inventarioID)
		return nil, NewError("conteo_bloqueado", "Solo el responsable del conteo puede acceder a él")
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

func (s *ServiceImpl) Eliminar(ctx context.Context, inventarioID int64, userID int64, role string, userTiendaID *int64) error {
	s.logger.InfoContext(ctx, "inventario.eliminar: iniciando",
		"user_id", userID,
		"user_role", role,
		"inventario_id", inventarioID)

	// RBAC: Solo admin puede eliminar (RF-INV-05.2)
	if role != "admin" {
		s.logger.WarnContext(ctx, "inventario.eliminar: unauthorized role",
			"user_id", userID,
			"user_role", role)
		return NewError("sin_permiso", "Solo admins pueden eliminar conteos")
	}

	inv, err := s.repo.GetInventario(ctx, inventarioID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.eliminar: error obteniendo",
			"inventario_id", inventarioID,
			"error", err.Error())
		return err
	}

	if inv == nil {
		s.logger.WarnContext(ctx, "inventario.eliminar: not found",
			"inventario_id", inventarioID)
		return NewError("not_found", "inventario no encontrado")
	}

	if inv.Estado != EstadoEnProgreso {
		s.logger.WarnContext(ctx, "inventario.eliminar: invalid estado",
			"inventario_id", inventarioID,
			"estado", inv.Estado)
		return NewError("eliminacion_no_permitida", "solo se pueden eliminar conteos en progreso")
	}

	return s.repo.DeleteInventario(ctx, inventarioID)
}

func (s *ServiceImpl) Sugerir(ctx context.Context) (*SugerenciaResp, error) {
	now := time.Now()
	hour := now.Hour()

	// Sugerir tipo y horario según hora del día (Colombia) per RF-INV-01.2
	// 06:00-10:59 → diario/apertura
	// 11:00-14:59 → diario/mediodía
	// 15:00-23:59 → diario/cierre
	var tipo Tipo = TipoDiario
	var horario Horario

	if hour >= 6 && hour < 11 {
		horario = HorarioApertura
	} else if hour >= 11 && hour < 15 {
		horario = HorarioMediodia
	} else if hour >= 15 {
		horario = HoriarioCierre
	} else {
		// Madrugada (00:00-05:59): sugerir diario/cierre como fallback
		horario = HoriarioCierre
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

// GetEstadoInventarioActivo verifica si hay un conteo activo en una tienda (T143)
// Retorna el conteo activo si existe, nil si no hay
func (s *ServiceImpl) GetEstadoInventarioActivo(ctx context.Context, tiendaID int64) (*InventarioResp, error) {
	s.logger.InfoContext(ctx, "inventario.estado: verificando",
		"tienda_id", tiendaID)

	// Usar CanRecordMovimiento para verificar si hay conteo activo
	canRecord, activeCountID, err := s.repo.CanRecordMovimiento(ctx, tiendaID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.estado: error verificando",
			"tienda_id", tiendaID,
			"error", err.Error())
		return nil, err
	}

	// Si no hay conteo activo, retornar nil
	if canRecord {
		return nil, nil
	}

	// Obtener detalles del conteo activo
	if activeCountID == nil {
		return nil, nil
	}

	inv, err := s.repo.GetInventarioDetalle(ctx, *activeCountID)
	if err != nil {
		s.logger.ErrorContext(ctx, "inventario.estado: error obteniendo detalles",
			"tienda_id", tiendaID,
			"inventario_id", activeCountID,
			"error", err.Error())
		return nil, err
	}

	s.logger.InfoContext(ctx, "inventario.estado: conteo activo encontrado",
		"tienda_id", tiendaID,
		"inventario_id", activeCountID)

	return s.mapInventarioToResp(inv), nil
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
