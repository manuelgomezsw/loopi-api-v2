package unidades_medida

import (
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

type cachedUMRepository struct {
	inner       UMRepository
	listCache   *cache.EntityCache[ListarUMResponse]
	byIDCache   *cache.EntityCache[UnidadMedida]
	detailCache *cache.EntityCache[DetalleUMResponse]
}

// NewCachedRepository envuelve inner con una capa de caché Ristretto de TTL configurable.
func NewCachedRepository(inner UMRepository, ttl time.Duration) (UMRepository, error) {
	listC, err := cache.New[ListarUMResponse](ttl)
	if err != nil {
		return nil, err
	}
	byIDC, err := cache.New[UnidadMedida](ttl)
	if err != nil {
		return nil, err
	}
	detailC, err := cache.New[DetalleUMResponse](ttl)
	if err != nil {
		return nil, err
	}
	return &cachedUMRepository{
		inner:       inner,
		listCache:   listC,
		byIDCache:   byIDC,
		detailCache: detailC,
	}, nil
}

// --- Lecturas cacheadas ---

func (r *cachedUMRepository) Listar(params *ListarUMParams) (*ListarUMResponse, error) {
	key := listarKey(params)
	val, err := cache.ReadThrough(r.listCache, key, func() (ListarUMResponse, error) {
		resp, e := r.inner.Listar(params)
		if e != nil {
			return ListarUMResponse{}, e
		}
		return *resp, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedUMRepository) ObtenerPorID(id uint64) (*UnidadMedida, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.byIDCache, key, func() (UnidadMedida, error) {
		u, e := r.inner.ObtenerPorID(id)
		if e != nil {
			return UnidadMedida{}, e
		}
		return *u, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedUMRepository) ObtenerPorIDConItems(id uint64) (*DetalleUMResponse, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.detailCache, key, func() (DetalleUMResponse, error) {
		d, e := r.inner.ObtenerPorIDConItems(id)
		if e != nil {
			return DetalleUMResponse{}, e
		}
		return *d, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// --- Lecturas no cacheadas (datos volátiles o transversales) ---

func (r *cachedUMRepository) ExisteConCodigo(codigo string) (bool, error) {
	return r.inner.ExisteConCodigo(codigo)
}

func (r *cachedUMRepository) ContarItemsConUnidadCanonica(id uint64) (int, error) {
	return r.inner.ContarItemsConUnidadCanonica(id)
}

func (r *cachedUMRepository) ContarUnidadesActivasPorTipo(tipo string, excludeID uint64) (int, error) {
	return r.inner.ContarUnidadesActivasPorTipo(tipo, excludeID)
}

// --- Escrituras con invalidación ---

func (r *cachedUMRepository) Crear(req *CrearUMRequest) (*UnidadMedida, error) {
	u, err := r.inner.Crear(req)
	if err != nil {
		return nil, err
	}
	r.listCache.Clear()
	return u, nil
}

func (r *cachedUMRepository) Editar(id uint64, req *EditarUMRequest) (*UnidadMedida, error) {
	u, err := r.inner.Editar(id, req)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("id:%d", id)
	r.byIDCache.Delete(key)
	r.detailCache.Delete(key)
	r.listCache.Clear()
	return u, nil
}

func (r *cachedUMRepository) Inactivar(id uint64) error {
	if err := r.inner.Inactivar(id); err != nil {
		return err
	}
	key := fmt.Sprintf("id:%d", id)
	r.byIDCache.Delete(key)
	r.detailCache.Delete(key)
	r.listCache.Clear()
	return nil
}

// listarKey construye una clave de caché determinista a partir de los params de Listar.
func listarKey(p *ListarUMParams) string {
	activo := "all"
	if p.Activo != nil {
		activo = fmt.Sprintf("%v", *p.Activo)
	}
	return fmt.Sprintf("list:tipo=%s:activo=%s:page=%d:limit=%d", p.Tipo, activo, p.Page, p.Limit)
}
