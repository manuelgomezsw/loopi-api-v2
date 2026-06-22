package categorias

import (
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

type cachedRepository struct {
	inner      Repository
	catCache   *cache.EntityCache[CatalogoResponse]
	byIDCache  *cache.EntityCache[Categoria]
}

// NewCachedRepository envuelve inner con caché Ristretto TTL 24 h.
// catCache almacena el catálogo completo; byIDCache almacena categorías individuales.
func NewCachedRepository(inner Repository, ttl time.Duration) (Repository, error) {
	catC, err := cache.New[CatalogoResponse](ttl)
	if err != nil {
		return nil, err
	}
	byIDC, err := cache.New[Categoria](ttl)
	if err != nil {
		return nil, err
	}
	return &cachedRepository{inner: inner, catCache: catC, byIDCache: byIDC}, nil
}

// --- Lecturas cacheadas ---

func (r *cachedRepository) ListarConItems(soloActivas *bool) (*CatalogoResponse, error) {
	key := catalogKey(soloActivas)
	val, err := cache.ReadThrough(r.catCache, key, func() (CatalogoResponse, error) {
		resp, e := r.inner.ListarConItems(soloActivas)
		if e != nil {
			return CatalogoResponse{}, e
		}
		return *resp, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedRepository) ObtenerCategoriaPorID(id uint64) (*Categoria, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.byIDCache, key, func() (Categoria, error) {
		c, e := r.inner.ObtenerCategoriaPorID(id)
		if e != nil {
			return Categoria{}, e
		}
		return *c, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// --- Lecturas no cacheadas ---

func (r *cachedRepository) ContarSubcategoriasActivas(categoriaID uint64) (int, error) {
	return r.inner.ContarSubcategoriasActivas(categoriaID)
}

func (r *cachedRepository) ObtenerSubcategoriaPorID(id uint64) (*Subcategoria, error) {
	return r.inner.ObtenerSubcategoriaPorID(id)
}

func (r *cachedRepository) ContarItemsPorSubcategoria(subcategoriaID uint64) (int, error) {
	return r.inner.ContarItemsPorSubcategoria(subcategoriaID)
}

// --- Escrituras con invalidación ---

func (r *cachedRepository) InsertarCategoria(nombre string, creadoPor uint64) (*Categoria, error) {
	c, err := r.inner.InsertarCategoria(nombre, creadoPor)
	if err != nil {
		return nil, err
	}
	r.catCache.Clear()
	return c, nil
}

func (r *cachedRepository) ActualizarCategoria(id uint64, nombre string, actualizadoPor uint64) (*Categoria, error) {
	c, err := r.inner.ActualizarCategoria(id, nombre, actualizadoPor)
	if err != nil {
		return nil, err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.catCache.Clear()
	return c, nil
}

func (r *cachedRepository) InactivarCategoria(id, actualizadoPor uint64) error {
	if err := r.inner.InactivarCategoria(id, actualizadoPor); err != nil {
		return err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.catCache.Clear()
	return nil
}

func (r *cachedRepository) InactivarSubcategoriasDeCategoria(categoriaID, actualizadoPor uint64) (int, error) {
	n, err := r.inner.InactivarSubcategoriasDeCategoria(categoriaID, actualizadoPor)
	if err != nil {
		return 0, err
	}
	r.catCache.Clear()
	return n, nil
}

func (r *cachedRepository) ReactivarCategoria(id, actualizadoPor uint64) (*Categoria, error) {
	c, err := r.inner.ReactivarCategoria(id, actualizadoPor)
	if err != nil {
		return nil, err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.catCache.Clear()
	return c, nil
}

func (r *cachedRepository) InsertarSubcategoria(nombre string, categoriaID, creadoPor uint64) (*Subcategoria, error) {
	s, err := r.inner.InsertarSubcategoria(nombre, categoriaID, creadoPor)
	if err != nil {
		return nil, err
	}
	r.catCache.Clear()
	return s, nil
}

func (r *cachedRepository) ActualizarSubcategoria(id uint64, nombre string, actualizadoPor uint64) (*Subcategoria, error) {
	s, err := r.inner.ActualizarSubcategoria(id, nombre, actualizadoPor)
	if err != nil {
		return nil, err
	}
	r.catCache.Clear()
	return s, nil
}

func (r *cachedRepository) InactivarSubcategoria(id, actualizadoPor uint64) error {
	if err := r.inner.InactivarSubcategoria(id, actualizadoPor); err != nil {
		return err
	}
	r.catCache.Clear()
	return nil
}

func (r *cachedRepository) ReactivarSubcategoria(id, actualizadoPor uint64) (*Subcategoria, error) {
	s, err := r.inner.ReactivarSubcategoria(id, actualizadoPor)
	if err != nil {
		return nil, err
	}
	r.catCache.Clear()
	return s, nil
}

func catalogKey(soloActivas *bool) string {
	if soloActivas == nil {
		return "list:todos"
	}
	return fmt.Sprintf("list:activo=%v", *soloActivas)
}
