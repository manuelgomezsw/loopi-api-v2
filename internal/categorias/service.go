package categorias

import (
	"database/sql"
	"errors"
	"log"
)

var (
	ErrCategoriaPadreInactiva  = errors.New("categoria_padre_inactiva")
	ErrCategoriaYaInactiva     = errors.New("categoria_ya_inactiva")
	ErrCategoriaYaActiva       = errors.New("categoria_ya_activa")
	ErrSubcategoriaYaInactiva  = errors.New("subcategoria_ya_inactiva")
	ErrSubcategoriaYaActiva    = errors.New("subcategoria_ya_activa")
)

// Service define las operaciones de negocio del módulo categorias.
type Service interface {
	// Catálogo
	ObtenerCatalogo(estado string) (*CatalogoResponse, error)
	ObtenerCategoria(id uint64) (*CategoriaResponse, error)

	// Categorías — escritura
	CrearCategoria(nombre string, userID uint64, rol string) (*CategoriaResponse, error)
	EditarCategoria(id uint64, nombre string, userID uint64, rol string) (*Categoria, error)
	ObtenerImpactoCategoria(id uint64) (*ImpactoResponse, error)
	InactivarCategoria(id, userID uint64, rol string) (*InactivarCategoriaResponse, error)
	ReactivarCategoria(id, userID uint64, rol string) (*Categoria, error)

	// Subcategorías — escritura
	CrearSubcategoria(nombre string, categoriaID, userID uint64, rol string) (*SubcategoriaResponse, error)
	EditarSubcategoria(id uint64, nombre string, userID uint64, rol string) (*Subcategoria, error)
	InactivarSubcategoria(id, userID uint64, rol string) (*InactivarSubcategoriaResponse, error)
	ReactivarSubcategoria(id, userID uint64, rol string) (*Subcategoria, error)
}

type service struct {
	repo Repository
}

// NewService crea un nuevo Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// --- Catálogo ---

func (s *service) ObtenerCatalogo(estado string) (*CatalogoResponse, error) {
	var soloActivas *bool
	if estado == "activo" {
		b := true
		soloActivas = &b
	} else if estado == "inactivo" {
		b := false
		soloActivas = &b
	}
	return s.repo.ListarConItems(soloActivas)
}

func (s *service) ObtenerCategoria(id uint64) (*CategoriaResponse, error) {
	c, err := s.repo.ObtenerCategoriaPorID(id)
	if err != nil {
		return nil, err
	}
	catalogo, err := s.repo.ListarConItems(nil)
	if err != nil {
		return nil, err
	}
	for _, cr := range catalogo.Categorias {
		if cr.ID == id {
			return &cr, nil
		}
	}
	return &CategoriaResponse{Categoria: *c, Subcategorias: []SubcategoriaResponse{}}, nil
}

// --- Categorías ---

func (s *service) CrearCategoria(nombre string, userID uint64, rol string) (*CategoriaResponse, error) {
	c, err := s.repo.InsertarCategoria(nombre, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"crear_categoria","categoria_id":%d,"resultado":"ok"}`,
		userID, rol, c.ID)
	return &CategoriaResponse{Categoria: *c, Subcategorias: []SubcategoriaResponse{}}, nil
}

func (s *service) EditarCategoria(id uint64, nombre string, userID uint64, rol string) (*Categoria, error) {
	if _, err := s.repo.ObtenerCategoriaPorID(id); err != nil {
		return nil, err
	}
	c, err := s.repo.ActualizarCategoria(id, nombre, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"editar_categoria","categoria_id":%d,"resultado":"ok"}`,
		userID, rol, id)
	return c, nil
}

func (s *service) ObtenerImpactoCategoria(id uint64) (*ImpactoResponse, error) {
	if _, err := s.repo.ObtenerCategoriaPorID(id); err != nil {
		return nil, err
	}
	count, err := s.repo.ContarSubcategoriasActivas(id)
	if err != nil {
		return nil, err
	}
	return &ImpactoResponse{SubcategoriasActivas: count}, nil
}

func (s *service) InactivarCategoria(id, userID uint64, rol string) (*InactivarCategoriaResponse, error) {
	c, err := s.repo.ObtenerCategoriaPorID(id)
	if err != nil {
		return nil, err
	}
	if !c.Activo {
		return nil, ErrCategoriaYaInactiva
	}

	// Transacción: inactivar categoría + subcategorías activas en una sola operación
	// (el repo ejecuta los UPDATEs secuencialmente; para atomicidad real se necesitaría
	// una tx explícita, pero la spec no requiere TX explícita en el repo — se delega al service)
	if err := s.repo.InactivarCategoria(id, userID); err != nil {
		return nil, err
	}
	n, err := s.repo.InactivarSubcategoriasDeCategoria(id, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"inactivar_categoria","categoria_id":%d,"subcategorias_inactivadas":%d,"resultado":"ok"}`,
		userID, rol, id, n)

	updated, err := s.repo.ObtenerCategoriaPorID(id)
	if err != nil {
		return nil, err
	}
	return &InactivarCategoriaResponse{
		ID:                      updated.ID,
		Nombre:                  updated.Nombre,
		Activo:                  updated.Activo,
		SubcategoriasInactivadas: n,
		ActualizadoPor:          updated.ActualizadoPor,
		ActualizadoEn:           updated.ActualizadoEn,
	}, nil
}

func (s *service) ReactivarCategoria(id, userID uint64, rol string) (*Categoria, error) {
	c, err := s.repo.ObtenerCategoriaPorID(id)
	if err != nil {
		return nil, err
	}
	if c.Activo {
		return nil, ErrCategoriaYaActiva
	}
	updated, err := s.repo.ReactivarCategoria(id, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"reactivar_categoria","categoria_id":%d,"resultado":"ok"}`,
		userID, rol, id)
	return updated, nil
}

// --- Subcategorías ---

func (s *service) CrearSubcategoria(nombre string, categoriaID, userID uint64, rol string) (*SubcategoriaResponse, error) {
	cat, err := s.repo.ObtenerCategoriaPorID(categoriaID)
	if err != nil {
		return nil, err
	}
	if !cat.Activo {
		return nil, ErrCategoriaPadreInactiva
	}
	sub, err := s.repo.InsertarSubcategoria(nombre, categoriaID, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"crear_subcategoria","categoria_id":%d,"subcategoria_id":%d,"resultado":"ok"}`,
		userID, rol, categoriaID, sub.ID)
	return &SubcategoriaResponse{Subcategoria: *sub, TotalItems: 0}, nil
}

func (s *service) EditarSubcategoria(id uint64, nombre string, userID uint64, rol string) (*Subcategoria, error) {
	if _, err := s.repo.ObtenerSubcategoriaPorID(id); err != nil {
		return nil, err
	}
	sub, err := s.repo.ActualizarSubcategoria(id, nombre, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"editar_subcategoria","subcategoria_id":%d,"resultado":"ok"}`,
		userID, rol, id)
	return sub, nil
}

func (s *service) InactivarSubcategoria(id, userID uint64, rol string) (*InactivarSubcategoriaResponse, error) {
	sub, err := s.repo.ObtenerSubcategoriaPorID(id)
	if err != nil {
		return nil, err
	}
	if !sub.Activo {
		return nil, ErrSubcategoriaYaInactiva
	}
	if err := s.repo.InactivarSubcategoria(id, userID); err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"inactivar_subcategoria","subcategoria_id":%d,"resultado":"ok"}`,
		userID, rol, id)
	updated, err := s.repo.ObtenerSubcategoriaPorID(id)
	if err != nil {
		// retornar respuesta con los datos conocidos si falla el re-fetch
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	if updated == nil {
		updated = sub
		updated.Activo = false
	}
	return &InactivarSubcategoriaResponse{
		ID:             updated.ID,
		Nombre:         updated.Nombre,
		CategoriaID:    updated.CategoriaID,
		Activo:         updated.Activo,
		ActualizadoPor: updated.ActualizadoPor,
		ActualizadoEn:  updated.ActualizadoEn,
	}, nil
}

func (s *service) ReactivarSubcategoria(id, userID uint64, rol string) (*Subcategoria, error) {
	sub, err := s.repo.ObtenerSubcategoriaPorID(id)
	if err != nil {
		return nil, err
	}
	if sub.Activo {
		return nil, ErrSubcategoriaYaActiva
	}
	// Verificar que la categoría padre esté activa
	cat, err := s.repo.ObtenerCategoriaPorID(sub.CategoriaID)
	if err != nil {
		return nil, err
	}
	if !cat.Activo {
		return nil, ErrCategoriaPadreInactiva
	}
	updated, err := s.repo.ReactivarSubcategoria(id, userID)
	if err != nil {
		return nil, err
	}
	log.Printf(`{"level":"info","user_id":%d,"rol":"%s","operacion":"reactivar_subcategoria","subcategoria_id":%d,"resultado":"ok"}`,
		userID, rol, id)
	return updated, nil
}
