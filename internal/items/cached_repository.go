package items

import (
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

type cachedRepository struct {
	inner       Repository
	listCache   *cache.EntityCache[ListarItemsResponse]
	byIDCache   *cache.EntityCache[Item]
	detailCache *cache.EntityCache[ItemConNombres]
}

// NewCachedRepository envuelve inner con una capa de caché Ristretto de TTL configurable.
func NewCachedRepository(inner Repository, ttl time.Duration) (Repository, error) {
	listC, err := cache.New[ListarItemsResponse](ttl)
	if err != nil {
		return nil, err
	}
	byIDC, err := cache.New[Item](ttl)
	if err != nil {
		return nil, err
	}
	detailC, err := cache.New[ItemConNombres](ttl)
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

func (r *cachedRepository) Listar(filtros *FiltrosListado) (*ListarItemsResponse, error) {
	key := listarKey(filtros)
	val, err := cache.ReadThrough(r.listCache, key, func() (ListarItemsResponse, error) {
		resp, e := r.inner.Listar(filtros)
		if e != nil {
			return ListarItemsResponse{}, e
		}
		return *resp, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedRepository) ObtenerPorID(id uint64) (*Item, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.byIDCache, key, func() (Item, error) {
		it, e := r.inner.ObtenerPorID(id)
		if e != nil {
			return Item{}, e
		}
		return *it, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedRepository) ObtenerDetallePorID(id uint64) (*ItemConNombres, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.detailCache, key, func() (ItemConNombres, error) {
		d, e := r.inner.ObtenerDetallePorID(id)
		if e != nil {
			return ItemConNombres{}, e
		}
		return *d, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

// --- Lecturas no cacheadas (datos volátiles, transversales o de baja frecuencia) ---

func (r *cachedRepository) ExisteCodigo(codigo string) (bool, error) {
	return r.inner.ExisteCodigo(codigo)
}

func (r *cachedRepository) ExisteNombre(nombre string, excludeID *uint64) (bool, error) {
	return r.inner.ExisteNombre(nombre, excludeID)
}

func (r *cachedRepository) EstaEnUso(itemID uint64) (bool, error) {
	return r.inner.EstaEnUso(itemID)
}

func (r *cachedRepository) VerificarSubcategoria(id uint64) (existe, activo bool, err error) {
	return r.inner.VerificarSubcategoria(id)
}

func (r *cachedRepository) VerificarProveedor(id uint64) (existe, activo bool, err error) {
	return r.inner.VerificarProveedor(id)
}

func (r *cachedRepository) VerificarUnidadMedida(id uint64) (existe, activo bool, err error) {
	return r.inner.VerificarUnidadMedida(id)
}

func (r *cachedRepository) VerificarTienda(id uint64) (existe, activo bool, err error) {
	return r.inner.VerificarTienda(id)
}

func (r *cachedRepository) ListarCostosTienda(itemID uint64) ([]CostoPorTienda, error) {
	return r.inner.ListarCostosTienda(itemID)
}

// --- Escrituras con invalidación ---

func (r *cachedRepository) Crear(req *CrearItemRequest, userID uint64) (*Item, error) {
	it, err := r.inner.Crear(req, userID)
	if err != nil {
		return nil, err
	}
	r.listCache.Clear()
	return it, nil
}

func (r *cachedRepository) Actualizar(id uint64, req *EditarItemRequest, userID uint64) (*Item, error) {
	it, err := r.inner.Actualizar(id, req, userID)
	if err != nil {
		return nil, err
	}
	r.invalidarItem(id)
	return it, nil
}

func (r *cachedRepository) CambiarEstado(id uint64, activo bool, userID uint64) (*Item, error) {
	it, err := r.inner.CambiarEstado(id, activo, userID)
	if err != nil {
		return nil, err
	}
	r.invalidarItem(id)
	return it, nil
}

func (r *cachedRepository) InsertarCostoTienda(itemID uint64, req *RegistrarCostoTiendaRequest, userID uint64) (*ItemCostoTienda, error) {
	return r.inner.InsertarCostoTienda(itemID, req, userID)
}

func (r *cachedRepository) invalidarItem(id uint64) {
	key := fmt.Sprintf("id:%d", id)
	r.byIDCache.Delete(key)
	r.detailCache.Delete(key)
	r.listCache.Clear()
}

// listarKey construye una clave de caché determinista a partir de los filtros de Listar.
func listarKey(f *FiltrosListado) string {
	activo := "all"
	if f.Activo != nil {
		activo = fmt.Sprintf("%v", *f.Activo)
	}
	return fmt.Sprintf("list:tipo=%s:frecuencia=%s:activo=%s:pagina=%d:por_pagina=%d",
		f.Tipo, f.Frecuencia, activo, f.Pagina, f.PorPagina)
}
