package proveedores

import (
	"errors"
	"log"
	"net/mail"
)

var (
	// ErrYaInactivo se retorna al intentar inactivar un proveedor ya inactivo.
	ErrYaInactivo = errors.New("ya_inactivo")
	// ErrYaActivo se retorna al intentar activar un proveedor ya activo.
	ErrYaActivo = errors.New("ya_activo")
)

// ValidationError es un error de validación con campo opcional.
type ValidationError struct {
	Codigo  string
	Mensaje string
	Campo   string
}

func (e *ValidationError) Error() string { return e.Codigo }

// Service define las operaciones de negocio del módulo de proveedores.
type Service interface {
	Crear(req *CrearProveedorRequest, userID uint64, rol string) (*Proveedor, error)
	Listar(filtros *FiltrosListado) (*ListarProveedoresResponse, error)
	ObtenerPorID(id uint64) (*ProveedorDetalleResponse, error)
	Editar(id uint64, req *EditarProveedorRequest, userID uint64, rol string) (*Proveedor, error)
	Inactivar(id, userID uint64, rol string) (*CambiarEstadoResponse, error)
	Activar(id, userID uint64, rol string) (*CambiarEstadoResponse, error)
}

type service struct {
	repo Repository
}

// NewService crea un nuevo Service. La caché se configura en el repositorio (decorador).
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func validarEmail(email *string) error {
	if email == nil || *email == "" {
		return nil
	}
	if _, err := mail.ParseAddress(*email); err != nil {
		return &ValidationError{Codigo: "email_invalido", Mensaje: "El formato del email no es válido.", Campo: "email_contacto"}
	}
	return nil
}

// Crear valida y persiste un nuevo proveedor.
func (s *service) Crear(req *CrearProveedorRequest, userID uint64, rol string) (*Proveedor, error) {
	if req.RazonSocial == "" {
		return nil, &ValidationError{Codigo: "campo_requerido", Mensaje: "La razón social es obligatoria.", Campo: "razon_social"}
	}
	if req.NIT == "" {
		return nil, &ValidationError{Codigo: "campo_requerido", Mensaje: "El NIT es obligatorio.", Campo: "nit"}
	}
	if err := validarEmail(req.EmailContacto); err != nil {
		return nil, err
	}

	existe, err := s.repo.ExisteNIT(req.NIT, nil)
	if err != nil {
		return nil, err
	}
	if existe {
		return nil, ErrNITDuplicado
	}

	p, err := s.repo.Crear(req)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"crear_proveedor","proveedor_id":%d}`,
		userID, rol, p.ID)
	return p, nil
}

// Listar retorna el catálogo paginado. La caché opera en el repositorio.
func (s *service) Listar(filtros *FiltrosListado) (*ListarProveedoresResponse, error) {
	return s.repo.Listar(filtros)
}

// ObtenerPorID retorna el detalle de un proveedor. La caché opera en el repositorio.
func (s *service) ObtenerPorID(id uint64) (*ProveedorDetalleResponse, error) {
	return s.repo.ObtenerPorIDConItems(id)
}

// Editar actualiza los campos enviados de un proveedor existente.
func (s *service) Editar(id uint64, req *EditarProveedorRequest, userID uint64, rol string) (*Proveedor, error) {
	if _, err := s.repo.ObtenerPorID(id); err != nil {
		return nil, err
	}

	if req.RazonSocial != nil && *req.RazonSocial == "" {
		return nil, &ValidationError{Codigo: "campo_vacio", Mensaje: "La razón social no puede quedar vacía.", Campo: "razon_social"}
	}
	if req.NIT != nil {
		if *req.NIT == "" {
			return nil, &ValidationError{Codigo: "campo_vacio", Mensaje: "El NIT no puede quedar vacío.", Campo: "nit"}
		}
		excludeID := id
		existe, err := s.repo.ExisteNIT(*req.NIT, &excludeID)
		if err != nil {
			return nil, err
		}
		if existe {
			return nil, ErrNITDuplicado
		}
	}
	if err := validarEmail(req.EmailContacto); err != nil {
		return nil, err
	}

	p, err := s.repo.Actualizar(id, req)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"editar_proveedor","proveedor_id":%d}`,
		userID, rol, id)
	return p, nil
}

// Inactivar marca el proveedor como inactivo si actualmente está activo.
func (s *service) Inactivar(id, userID uint64, rol string) (*CambiarEstadoResponse, error) {
	p, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	if !p.Activo {
		return nil, ErrYaInactivo
	}
	if err := s.repo.CambiarEstado(id, false); err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"inactivar_proveedor","proveedor_id":%d}`,
		userID, rol, id)
	return &CambiarEstadoResponse{ID: id, Activo: false, Mensaje: "Proveedor inactivado correctamente."}, nil
}

// Activar reactiva el proveedor si actualmente está inactivo.
func (s *service) Activar(id, userID uint64, rol string) (*CambiarEstadoResponse, error) {
	p, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	if p.Activo {
		return nil, ErrYaActivo
	}
	if err := s.repo.CambiarEstado(id, true); err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"activar_proveedor","proveedor_id":%d}`,
		userID, rol, id)
	return &CambiarEstadoResponse{ID: id, Activo: true, Mensaje: "Proveedor activado correctamente."}, nil
}
