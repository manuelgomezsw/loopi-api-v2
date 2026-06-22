package tiendas

import (
	"errors"
	"strings"
	"time"
)

// ErrTiendaYaInactiva se retorna al intentar inactivar una tienda que ya está inactiva.
var ErrTiendaYaInactiva = errors.New("tienda_ya_inactiva")

// ErrTiendaYaActiva se retorna al intentar reactivar una tienda que ya está activa.
var ErrTiendaYaActiva = errors.New("tienda_ya_activa")

// bogotaLoc es la zona horaria de América/Bogotá usada para todos los timestamps.
var bogotaLoc = func() *time.Location {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// TiendaService define las operaciones de negocio para tiendas.
type TiendaService interface {
	Listar(estado string, pagina, limite int) (ListaTiendasResponse, error)
	Crear(req TiendaRequest, adminID uint64) (TiendaResponse, error)
	ObtenerPorID(id uint64) (TiendaResponse, error)
	Actualizar(id uint64, req TiendaUpdateRequest, adminID uint64) (TiendaResponse, error)
	Inactivar(id, adminID uint64) (TiendaResponse, error)
	Reactivar(id, adminID uint64) (TiendaResponse, error)
}

type tiendaService struct {
	repo TiendaRepository
}

// NewService crea un nuevo TiendaService con inyección del repositorio.
func NewService(repo TiendaRepository) TiendaService {
	return &tiendaService{repo: repo}
}

// Listar retorna tiendas paginadas. Valida estado, pagina y limite.
func (s *tiendaService) Listar(estado string, pagina, limite int) (ListaTiendasResponse, error) {
	estadosValidos := map[string]bool{"todos": true, "activo": true, "inactivo": true}
	if !estadosValidos[estado] {
		return ListaTiendasResponse{}, &ValidationError{
			Codigo:  "estado_invalido",
			Mensaje: "El estado debe ser 'activo', 'inactivo' o 'todos'.",
			Campo:   "estado",
		}
	}
	if pagina < 1 {
		pagina = 1
	}
	if limite < 1 || limite > 100 {
		limite = 50
	}

	tiendas, total, err := s.repo.Listar(estado, pagina, limite)
	if err != nil {
		return ListaTiendasResponse{}, err
	}

	datos := make([]TiendaResponse, len(tiendas))
	for i, t := range tiendas {
		datos[i] = toResponse(t)
	}
	return ListaTiendasResponse{
		Datos:  datos,
		Total:  total,
		Pagina: pagina,
		Limite: limite,
	}, nil
}

// Crear normaliza el código a mayúsculas y persiste la tienda nueva.
func (s *tiendaService) Crear(req TiendaRequest, adminID uint64) (TiendaResponse, error) {
	ahora := time.Now().In(bogotaLoc)
	t := Tienda{
		Codigo:         strings.ToUpper(req.Codigo),
		Nombre:         req.Nombre,
		Direccion:      req.Direccion,
		Ciudad:         req.Ciudad,
		Telefono:       req.Telefono,
		CreadoPor:      adminID,
		CreadoEn:       ahora,
		ActualizadoPor: adminID,
		ActualizadoEn:  ahora,
	}
	created, err := s.repo.Crear(t)
	if err != nil {
		return TiendaResponse{}, err
	}
	return toResponse(created), nil
}

// ObtenerPorID retorna la tienda o ErrTiendaNoEncontrada.
func (s *tiendaService) ObtenerPorID(id uint64) (TiendaResponse, error) {
	t, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return TiendaResponse{}, err
	}
	return toResponse(t), nil
}

// Actualizar edita nombre, dirección, ciudad y teléfono. Ignora codigo.
func (s *tiendaService) Actualizar(id uint64, req TiendaUpdateRequest, adminID uint64) (TiendaResponse, error) {
	if _, err := s.repo.ObtenerPorID(id); err != nil {
		return TiendaResponse{}, err
	}
	ahora := time.Now().In(bogotaLoc)
	updated, err := s.repo.Actualizar(id, req, adminID, ahora)
	if err != nil {
		return TiendaResponse{}, err
	}
	return toResponse(updated), nil
}

// Inactivar pone activo=false. Retorna ErrTiendaYaInactiva si ya está inactiva.
func (s *tiendaService) Inactivar(id, adminID uint64) (TiendaResponse, error) {
	tienda, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return TiendaResponse{}, err
	}
	if !tienda.Activo {
		return TiendaResponse{}, ErrTiendaYaInactiva
	}
	ahora := time.Now().In(bogotaLoc)
	updated, err := s.repo.CambiarActivo(id, false, adminID, ahora)
	if err != nil {
		return TiendaResponse{}, err
	}
	return toResponse(updated), nil
}

// Reactivar pone activo=true. Retorna ErrTiendaYaActiva si ya está activa.
func (s *tiendaService) Reactivar(id, adminID uint64) (TiendaResponse, error) {
	tienda, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return TiendaResponse{}, err
	}
	if tienda.Activo {
		return TiendaResponse{}, ErrTiendaYaActiva
	}
	ahora := time.Now().In(bogotaLoc)
	updated, err := s.repo.CambiarActivo(id, true, adminID, ahora)
	if err != nil {
		return TiendaResponse{}, err
	}
	return toResponse(updated), nil
}

// ValidationError es un error de validación de input con campo opcional.
type ValidationError struct {
	Codigo  string
	Mensaje string
	Campo   string
}

func (e *ValidationError) Error() string { return e.Codigo }
