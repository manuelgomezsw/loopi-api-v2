package unidades_medida

import (
	"errors"
	"fmt"
	"log"
)

var (
	ErrTipoInvalido            = errors.New("tipo_invalido")
	ErrFactorInvalido          = errors.New("factor_invalido")
	ErrYaInactiva              = errors.New("ya_inactiva")
	ErrUnidadBaseNoInactivable = errors.New("unidad_base_no_inactivable")
	ErrFactorBaseInmutable     = errors.New("factor_base_inmutable")
)

// UMService define las operaciones de negocio del módulo de unidades de medida.
type UMService interface {
	Crear(req *CrearUMRequest, userID uint64, rol string) (*UnidadMedida, error)
	Inactivar(id, userID uint64, rol string) (*InactivarResponse, error)
	ObtenerImpacto(id uint64) (*ImpactoResponse, error)
	Listar(params *ListarUMParams) (*ListarUMResponse, error)
	ObtenerPorID(id uint64) (*DetalleUMResponse, error)
	Editar(id uint64, req *EditarUMRequest, userID uint64, rol string) (*UnidadMedida, error)
}

type umService struct {
	repo UMRepository
}

// NewService crea un nuevo UMService. La caché se configura en el repositorio (decorador).
func NewService(repo UMRepository) UMService {
	return &umService{repo: repo}
}

var tiposValidos = map[string]bool{"peso": true, "volumen": true, "unidad": true}

// Crear valida y persiste una nueva unidad de medida.
func (s *umService) Crear(req *CrearUMRequest, userID uint64, rol string) (*UnidadMedida, error) {
	if !tiposValidos[req.TipoMedida] {
		return nil, ErrTipoInvalido
	}
	if req.FactorConversion <= 0 {
		return nil, ErrFactorInvalido
	}
	existe, err := s.repo.ExisteConCodigo(req.Codigo)
	if err != nil {
		return nil, err
	}
	if existe {
		return nil, ErrCodigoDuplicado
	}
	u, err := s.repo.Crear(req)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"crear_unidad","unidad_id":%d}`,
		userID, rol, u.ID)
	return u, nil
}

// Inactivar desactiva la unidad si no es base activa ni está ya inactiva.
func (s *umService) Inactivar(id, userID uint64, rol string) (*InactivarResponse, error) {
	u, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	if !u.Activo {
		return nil, ErrYaInactiva
	}
	if u.UnidadBase {
		count, err := s.repo.ContarUnidadesActivasPorTipo(u.TipoMedida, id)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, ErrUnidadBaseNoInactivable
		}
	}
	if err := s.repo.Inactivar(id); err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"inactivar_unidad","unidad_id":%d}`,
		userID, rol, id)
	return &InactivarResponse{ID: id, Activo: false, Mensaje: "Unidad inactivada correctamente."}, nil
}

// ObtenerImpacto retorna cuántos items activos usan esta unidad como canónica.
func (s *umService) ObtenerImpacto(id uint64) (*ImpactoResponse, error) {
	u, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.ContarItemsConUnidadCanonica(u.ID)
	if err != nil {
		return nil, err
	}
	resp := &ImpactoResponse{UnidadID: id, ItemsConUnidadCanonica: count}
	if count > 0 {
		msg := fmt.Sprintf(
			"Al inactivar esta unidad, %d item(s) quedarán con unidad canónica inactiva y sus transacciones nuevas serán bloqueadas hasta que se les reasigne una unidad activa.",
			count,
		)
		resp.Advertencia = &msg
	}
	return resp, nil
}

// Listar retorna el catálogo paginado. La caché opera en el repositorio.
func (s *umService) Listar(params *ListarUMParams) (*ListarUMResponse, error) {
	return s.repo.Listar(params)
}

// ObtenerPorID retorna el detalle de una unidad. La caché opera en el repositorio.
func (s *umService) ObtenerPorID(id uint64) (*DetalleUMResponse, error) {
	return s.repo.ObtenerPorIDConItems(id)
}

// Editar actualiza nombre y/o factor_conversion con validaciones de negocio.
func (s *umService) Editar(id uint64, req *EditarUMRequest, userID uint64, rol string) (*UnidadMedida, error) {
	u, err := s.repo.ObtenerPorID(id)
	if err != nil {
		return nil, err
	}
	if req.FactorConversion != nil {
		if u.UnidadBase {
			return nil, ErrFactorBaseInmutable
		}
		if *req.FactorConversion <= 0 {
			return nil, ErrFactorInvalido
		}
	}
	updated, err := s.repo.Editar(id, req)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"editar_unidad","unidad_id":%d}`,
		userID, rol, id)
	return updated, nil
}
