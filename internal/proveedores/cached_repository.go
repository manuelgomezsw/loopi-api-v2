package proveedores

import (
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

type cachedRepository struct {
	inner       Repository
	listCache   *cache.EntityCache[ListarProveedoresResponse]
	byIDCache   *cache.EntityCache[Proveedor]
	detailCache *cache.EntityCache[ProveedorDetalleResponse]
}

// NewCachedRepository envuelve inner con una capa de caché Ristretto de TTL configurable.
func NewCachedRepository(inner Repository, ttl time.Duration) (Repository, error) {
	listC, err := cache.New[ListarProveedoresResponse](ttl)
	if err != nil {
		return nil, err
	}
	byIDC, err := cache.New[Proveedor](ttl)
	if err != nil {
		return nil, err
	}
	detailC, err := cache.New[ProveedorDetalleResponse](ttl)
	if err != nil {
		return nil, err
	}
	return &cachedRepository{
		inner:       inner,
		listCache:   listC,
		byIDCache:   byIDC,
		detailCache: detailC,
	}, nil
}

// --- Lecturas cacheadas ---

func (r *cachedRepository) Listar(filtros *FiltrosListado) (*ListarProveedoresResponse, error) {
	key := listarKey(filtros)
	val, err := cache.ReadThrough(r.listCache, key, func() (ListarProveedoresResponse, error) {
		resp, e := r.inner.Listar(filtros)
		if e != nil {
			return ListarProveedoresResponse{}, e
		}
		return *resp, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedRepository) ObtenerPorID(id uint64) (*Proveedor, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.byIDCache, key, func() (Proveedor, error) {
		p, e := r.inner.ObtenerPorID(id)
		if e != nil {
			return Proveedor{}, e
		}
		return *p, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedRepository) ObtenerPorIDConItems(id uint64) (*ProveedorDetalleResponse, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.detailCache, key, func() (ProveedorDetalleResponse, error) {
		d, e := r.inner.ObtenerPorIDConItems(id)
		if e != nil {
			return ProveedorDetalleResponse{}, e
		}
		return *d, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// --- Lecturas no cacheadas (datos volátiles o transversales) ---

func (r *cachedRepository) ExisteNIT(nit string, excludeID *uint64) (bool, error) {
	return r.inner.ExisteNIT(nit, excludeID)
}

func (r *cachedRepository) ContarItemsAsignados(id uint64) (int, error) {
	return r.inner.ContarItemsAsignados(id)
}

// --- Escrituras con invalidación ---

func (r *cachedRepository) Crear(req *CrearProveedorRequest) (*Proveedor, error) {
	p, err := r.inner.Crear(req)
	if err != nil {
		return nil, err
	}
	r.listCache.Clear()
	return p, nil
}

func (r *cachedRepository) Actualizar(id uint64, req *EditarProveedorRequest) (*Proveedor, error) {
	p, err := r.inner.Actualizar(id, req)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("id:%d", id)
	r.byIDCache.Delete(key)
	r.detailCache.Delete(key)
	r.listCache.Clear()
	return p, nil
}

func (r *cachedRepository) CambiarEstado(id uint64, activo bool) error {
	if err := r.inner.CambiarEstado(id, activo); err != nil {
		return err
	}
	key := fmt.Sprintf("id:%d", id)
	r.byIDCache.Delete(key)
	r.detailCache.Delete(key)
	r.listCache.Clear()
	return nil
}

// listarKey construye una clave de caché determinista a partir de los filtros de Listar.
func listarKey(f *FiltrosListado) string {
	activo := "all"
	if f.Activo != nil {
		activo = fmt.Sprintf("%v", *f.Activo)
	}
	return fmt.Sprintf("list:activo=%s:busqueda=%s:page=%d:limit=%d", activo, f.Busqueda, f.Page, f.Limit)
}
