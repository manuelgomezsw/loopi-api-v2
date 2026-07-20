package tiendas

import (
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

// listarTiendasResult agrupa el slice y el total para cachear Listar en un solo valor.
type listarTiendasResult struct {
	Items []Tienda
	Total int
}

type cachedTiendaRepository struct {
	inner     TiendaRepository
	listCache *cache.EntityCache[listarTiendasResult]
	byIDCache *cache.EntityCache[Tienda]
}

// NewCachedRepository envuelve inner con una capa de caché Ristretto de TTL configurable.
func NewCachedRepository(inner TiendaRepository, ttl time.Duration) (TiendaRepository, error) {
	listC, err := cache.New[listarTiendasResult](ttl)
	if err != nil {
		return nil, err
	}
	byIDC, err := cache.New[Tienda](ttl)
	if err != nil {
		return nil, err
	}
	return &cachedTiendaRepository{inner: inner, listCache: listC, byIDCache: byIDC}, nil
}

// --- Lecturas cacheadas ---

func (r *cachedTiendaRepository) ObtenerPorID(id uint64) (Tienda, error) {
	key := fmt.Sprintf("id:%d", id)
	return cache.ReadThrough(r.byIDCache, key, func() (Tienda, error) {
		return r.inner.ObtenerPorID(id)
	})
}

func (r *cachedTiendaRepository) Listar(estado string, pagina, limite int) ([]Tienda, int, error) {
	key := fmt.Sprintf("list:estado=%s:page=%d:limit=%d", estado, pagina, limite)
	res, err := cache.ReadThrough(r.listCache, key, func() (listarTiendasResult, error) {
		items, total, e := r.inner.Listar(estado, pagina, limite)
		if e != nil {
			return listarTiendasResult{}, e
		}
		return listarTiendasResult{Items: items, Total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return res.Items, res.Total, nil
}

// --- Escrituras con invalidación ---

func (r *cachedTiendaRepository) Crear(t Tienda) (Tienda, error) {
	result, err := r.inner.Crear(t)
	if err != nil {
		return Tienda{}, err
	}
	r.listCache.Clear()
	return result, nil
}

func (r *cachedTiendaRepository) Actualizar(id uint64, req TiendaUpdateRequest, adminID uint64, ahora time.Time) (Tienda, error) {
	result, err := r.inner.Actualizar(id, req, adminID, ahora)
	if err != nil {
		return Tienda{}, err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.listCache.Clear()
	return result, nil
}

func (r *cachedTiendaRepository) CambiarActivo(id uint64, activo bool, adminID uint64, ahora time.Time) (Tienda, error) {
	result, err := r.inner.CambiarActivo(id, activo, adminID, ahora)
	if err != nil {
		return Tienda{}, err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.listCache.Clear()
	return result, nil
}
