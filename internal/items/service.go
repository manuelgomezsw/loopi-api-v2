package items

import (
	"errors"
	"log"
	"strings"
)

var (
	// ErrItemYaInactivo se retorna al intentar inactivar un item ya inactivo.
	ErrItemYaInactivo = errors.New("item_ya_inactivo")
	// ErrItemYaActivo se retorna al intentar reactivar un item ya activo.
	ErrItemYaActivo = errors.New("item_ya_activo")
	// ErrCodigoEnUso se retorna al intentar cambiar el código de un item con usos registrados.
	ErrCodigoEnUso = errors.New("codigo_en_uso")
	// ErrCambioUnidadRequiereConfirmacion se retorna al cambiar la unidad de medida de un item
	// con historial de stock sin enviar confirmar_cambio_unidad=true.
	ErrCambioUnidadRequiereConfirmacion = errors.New("cambio_unidad_requiere_confirmacion")
	// ErrSubcategoriaNoEncontrada se retorna cuando la subcategoría referenciada no existe.
	ErrSubcategoriaNoEncontrada = errors.New("subcategoria_no_encontrada")
	// ErrSubcategoriaInactiva se retorna cuando la subcategoría referenciada está inactiva.
	ErrSubcategoriaInactiva = errors.New("subcategoria_inactiva")
	// ErrProveedorNoEncontrado se retorna cuando el proveedor referenciado no existe.
	ErrProveedorNoEncontrado = errors.New("proveedor_no_encontrado")
	// ErrProveedorInactivo se retorna cuando el proveedor referenciado está inactivo.
	ErrProveedorInactivo = errors.New("proveedor_inactivo")
	// ErrUnidadMedidaNoEncontrada se retorna cuando la unidad de medida referenciada no existe.
	ErrUnidadMedidaNoEncontrada = errors.New("unidad_medida_no_encontrada")
	// ErrUnidadMedidaInactiva se retorna cuando la unidad de medida referenciada está inactiva.
	ErrUnidadMedidaInactiva = errors.New("unidad_medida_inactiva")
	// ErrTiendaNoEncontrada se retorna cuando la tienda referenciada no existe.
	ErrTiendaNoEncontrada = errors.New("tienda_no_encontrada")
	// ErrTiendaInactiva se retorna cuando la tienda referenciada está inactiva.
	ErrTiendaInactiva = errors.New("tienda_inactiva")
)

var tiposValidos = map[string]bool{"insumo": true, "material_consumo": true, "activo": true}
var frecuenciasValidas = map[string]bool{"diario": true, "semanal": true, "mensual": true}

// ValidationError es un error de validación con campo opcional.
type ValidationError struct {
	Codigo  string
	Mensaje string
	Campo   string
}

func (e *ValidationError) Error() string { return e.Codigo }

// Service define las operaciones de negocio del módulo de items.
type Service interface {
	Crear(req *CrearItemRequest, userID uint64, rol string) (*ItemDetalleResponse, error)
	Listar(filtros *FiltrosListado) (*ListarItemsResponse, error)
	ObtenerPorID(id uint64) (*ItemDetalleResponse, error)
	Editar(id uint64, req *EditarItemRequest, userID uint64, rol string) (*ItemDetalleResponse, error)
	Inactivar(id, userID uint64, rol string) (*CambiarEstadoResponse, error)
	Reactivar(id, userID uint64, rol string) (*CambiarEstadoResponse, error)
	RegistrarCostoTienda(itemID uint64, req *RegistrarCostoTiendaRequest, userID uint64, rol string) (*ItemCostoTienda, error)
	ObtenerHistorialCostos(itemID uint64) (*HistorialCostosResponse, error)
}

type service struct {
	repo Repository
}

// NewService crea un nuevo Service. La caché se configura en el repositorio (decorador).
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// tieneHistorialStock indica si el item tiene historial de stock previo en la unidad actual.
// Retorna false hasta que 009-inventario-conteo exista — ver specs/009-inventario-conteo/spec.md
// §Dependencias. TODO: reemplazar por consulta real a inventarios_conteos_items cuando 009 se
// implemente.
func tieneHistorialStock(itemID uint64) bool {
	return false
}

func validarCamposComunes(codigo *string, nombre, tipo *string, subcategoriaID uint64, unidadMedidaID uint64, frecuencia, stockSeguridad string) error {
	if codigo != nil && strings.TrimSpace(*codigo) == "" {
		return &ValidationError{Codigo: "codigo_requerido", Mensaje: "El código es obligatorio.", Campo: "codigo"}
	}
	if nombre != nil && strings.TrimSpace(*nombre) == "" {
		return &ValidationError{Codigo: "nombre_requerido", Mensaje: "El nombre es obligatorio.", Campo: "nombre"}
	}
	if tipo != nil && !tiposValidos[*tipo] {
		return &ValidationError{Codigo: "tipo_invalido", Mensaje: "El tipo debe ser insumo, material_consumo o activo.", Campo: "tipo"}
	}
	if subcategoriaID == 0 {
		return &ValidationError{Codigo: "subcategoria_requerida", Mensaje: "La subcategoría es obligatoria.", Campo: "subcategoria_id"}
	}
	if unidadMedidaID == 0 {
		return &ValidationError{Codigo: "unidad_medida_requerida", Mensaje: "La unidad de medida es obligatoria.", Campo: "unidad_medida_id"}
	}
	if !frecuenciasValidas[frecuencia] {
		return &ValidationError{Codigo: "frecuencia_invalida", Mensaje: "La frecuencia debe ser diario, semanal o mensual.", Campo: "frecuencia_inventario"}
	}
	if valor, err := parseStockSeguridad(stockSeguridad); err != nil || !valor {
		return &ValidationError{Codigo: "stock_seguridad_requerido", Mensaje: "El stock de seguridad es obligatorio y no puede ser negativo.", Campo: "stock_seguridad"}
	}
	return nil
}

// verificarReferencias valida que subcategoría, proveedor (si aplica) y unidad de medida
// existan y estén activos. Común a Crear y Editar.
func (s *service) verificarReferencias(subcategoriaID uint64, proveedorID *uint64, unidadMedidaID uint64) error {
	existe, activa, err := s.repo.VerificarSubcategoria(subcategoriaID)
	if err != nil {
		return err
	}
	if !existe {
		return ErrSubcategoriaNoEncontrada
	}
	if !activa {
		return ErrSubcategoriaInactiva
	}

	if proveedorID != nil {
		existe, activo, err := s.repo.VerificarProveedor(*proveedorID)
		if err != nil {
			return err
		}
		if !existe {
			return ErrProveedorNoEncontrado
		}
		if !activo {
			return ErrProveedorInactivo
		}
	}

	existe, activa, err = s.repo.VerificarUnidadMedida(unidadMedidaID)
	if err != nil {
		return err
	}
	if !existe {
		return ErrUnidadMedidaNoEncontrada
	}
	if !activa {
		return ErrUnidadMedidaInactiva
	}
	return nil
}

// Crear valida y persiste un nuevo item.
func (s *service) Crear(req *CrearItemRequest, userID uint64, rol string) (*ItemDetalleResponse, error) {
	codigo := req.Codigo
	nombre := req.Nombre
	tipo := req.Tipo
	if err := validarCamposComunes(&codigo, &nombre, &tipo, req.SubcategoriaID, req.UnidadMedidaID, req.FrecuenciaInventario, req.StockSeguridad); err != nil {
		return nil, err
	}

	if err := s.verificarReferencias(req.SubcategoriaID, req.ProveedorID, req.UnidadMedidaID); err != nil {
		return nil, err
	}

	existeCodigo, err := s.repo.ExisteCodigo(req.Codigo)
	if err != nil {
		return nil, err
	}
	if existeCodigo {
		return nil, ErrCodigoDuplicado
	}
	existeNombre, err := s.repo.ExisteNombre(req.Nombre, nil)
	if err != nil {
		return nil, err
	}
	if existeNombre {
		return nil, ErrNombreDuplicado
	}

	it, err := s.repo.Crear(req, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"crear_item","item_id":%d,"item_codigo":"%s","resultado":"ok"}`,
		userID, rol, it.ID, it.Codigo)

	detalle, err := s.repo.ObtenerDetallePorID(it.ID)
	if err != nil {
		return nil, err
	}
	return &ItemDetalleResponse{ItemConNombres: *detalle, EstaEnUso: false}, nil
}

// Listar retorna el catálogo paginado. La caché opera en el repositorio.
func (s *service) Listar(filtros *FiltrosListado) (*ListarItemsResponse, error) {
	return s.repo.Listar(filtros)
}

// ObtenerPorID retorna el detalle de un item, incluyendo si está en uso.
func (s *service) ObtenerPorID(id uint64) (*ItemDetalleResponse, error) {
	detalle, err := s.repo.ObtenerDetallePorID(id)
	if err != nil {
		return nil, err
	}
	enUso, err := s.repo.EstaEnUso(id)
	if err != nil {
		return nil, err
	}
	return &ItemDetalleResponse{ItemConNombres: *detalle, EstaEnUso: enUso}, nil
}

// Editar actualiza un item existente, con reemplazo completo de sus campos operativos.
func (s *service) Editar(id uint64, req *EditarItemRequest, userID uint64, rol string) (*ItemDetalleResponse, error) {
	actual, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}

	if err := validarCamposComunes(nil, &req.Nombre, nil, req.SubcategoriaID, req.UnidadMedidaID, req.FrecuenciaInventario, req.StockSeguridad); err != nil {
		return nil, err
	}

	if err := s.verificarReferencias(req.SubcategoriaID, req.ProveedorID, req.UnidadMedidaID); err != nil {
		return nil, err
	}

	// Código: solo se valida/bloquea si el request intenta cambiarlo.
	nuevoCodigo := actual.Codigo
	if req.Codigo != "" {
		nuevoCodigo = req.Codigo
	}
	if nuevoCodigo != actual.Codigo {
		enUso, err := s.repo.EstaEnUso(id)
		if err != nil {
			return nil, err
		}
		if enUso {
			return nil, ErrCodigoEnUso
		}
		existeCodigo, err := s.repo.ExisteCodigo(nuevoCodigo)
		if err != nil {
			return nil, err
		}
		if existeCodigo {
			return nil, ErrCodigoDuplicado
		}
	}
	req.Codigo = nuevoCodigo

	// Unidad de medida: cambiarla con historial de stock requiere confirmación explícita.
	if req.UnidadMedidaID != actual.UnidadMedidaID && tieneHistorialStock(id) && !req.ConfirmarCambioUnidad {
		return nil, ErrCambioUnidadRequiereConfirmacion
	}

	if req.Nombre != actual.Nombre {
		existeNombre, err := s.repo.ExisteNombre(req.Nombre, &id)
		if err != nil {
			return nil, err
		}
		if existeNombre {
			return nil, ErrNombreDuplicado
		}
	}

	if _, err := s.repo.Actualizar(id, req, userID); err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"editar_item","item_id":%d,"item_codigo":"%s","resultado":"ok"}`,
		userID, rol, id, nuevoCodigo)

	return s.ObtenerPorID(id)
}

// Inactivar marca el item como inactivo si actualmente está activo.
func (s *service) Inactivar(id, userID uint64, rol string) (*CambiarEstadoResponse, error) {
	it, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	if !it.Activo {
		return nil, ErrItemYaInactivo
	}
	updated, err := s.repo.CambiarEstado(id, false, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"inactivar_item","item_id":%d,"item_codigo":"%s","resultado":"ok"}`,
		userID, rol, id, updated.Codigo)
	return &CambiarEstadoResponse{
		ID: updated.ID, Codigo: updated.Codigo, Nombre: updated.Nombre, Activo: updated.Activo,
		ActualizadoPor: updated.ActualizadoPor, ActualizadoEn: updated.ActualizadoEn,
	}, nil
}

// Reactivar reactiva el item si actualmente está inactivo.
func (s *service) Reactivar(id, userID uint64, rol string) (*CambiarEstadoResponse, error) {
	it, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	if it.Activo {
		return nil, ErrItemYaActivo
	}
	updated, err := s.repo.CambiarEstado(id, true, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"reactivar_item","item_id":%d,"item_codigo":"%s","resultado":"ok"}`,
		userID, rol, id, updated.Codigo)
	return &CambiarEstadoResponse{
		ID: updated.ID, Codigo: updated.Codigo, Nombre: updated.Nombre, Activo: updated.Activo,
		ActualizadoPor: updated.ActualizadoPor, ActualizadoEn: updated.ActualizadoEn,
	}, nil
}

// RegistrarCostoTienda inserta un nuevo costo histórico para una tienda específica.
func (s *service) RegistrarCostoTienda(itemID uint64, req *RegistrarCostoTiendaRequest, userID uint64, rol string) (*ItemCostoTienda, error) {
	if _, err := s.repo.ObtenerPorID(itemID); err != nil {
		return nil, err
	}
	if req.TiendaID == 0 {
		return nil, &ValidationError{Codigo: "tienda_requerida", Mensaje: "La tienda es obligatoria.", Campo: "tienda_id"}
	}
	if req.CostoUnitario <= 0 {
		return nil, &ValidationError{Codigo: "costo_invalido", Mensaje: "El costo unitario debe ser mayor a cero.", Campo: "costo_unitario"}
	}

	existe, activa, err := s.repo.VerificarTienda(req.TiendaID)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, ErrTiendaNoEncontrada
	}
	if !activa {
		return nil, ErrTiendaInactiva
	}

	c, err := s.repo.InsertarCostoTienda(itemID, req, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"registrar_costo_tienda","item_id":%d,"tienda_id":%d,"resultado":"ok"}`,
		userID, rol, itemID, req.TiendaID)
	return c, nil
}

// ObtenerHistorialCostos retorna el historial de costos por tienda de un item.
func (s *service) ObtenerHistorialCostos(itemID uint64) (*HistorialCostosResponse, error) {
	it, err := s.repo.ObtenerPorID(itemID)
	if err != nil {
		return nil, err
	}
	costos, err := s.repo.ListarCostosTienda(itemID)
	if err != nil {
		return nil, err
	}
	return &HistorialCostosResponse{ItemID: itemID, CostoGlobal: it.CostoUnitario, CostosPorTienda: costos}, nil
}
