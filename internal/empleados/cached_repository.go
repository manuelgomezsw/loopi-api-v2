package empleados

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

// listarEmpleadosResult agrupa el slice y el total para cachear ListarEmpleados.
type listarEmpleadosResult struct {
	Items []Empleado
	Total int
}

type cachedEmpleadoRepository struct {
	inner     EmpleadoRepository
	byIDCache *cache.EntityCache[Empleado]
	listCache *cache.EntityCache[listarEmpleadosResult]
}

// NewCachedRepository envuelve inner con una capa de caché Ristretto de TTL configurable.
func NewCachedRepository(inner EmpleadoRepository, ttl time.Duration) (EmpleadoRepository, error) {
	byIDC, err := cache.New[Empleado](ttl)
	if err != nil {
		return nil, err
	}
	listC, err := cache.New[listarEmpleadosResult](ttl)
	if err != nil {
		return nil, err
	}
	return &cachedEmpleadoRepository{inner: inner, byIDCache: byIDC, listCache: listC}, nil
}

// --- Lecturas cacheadas ---

func (r *cachedEmpleadoRepository) ObtenerPorID(ctx context.Context, id uint64) (*Empleado, error) {
	key := fmt.Sprintf("id:%d", id)
	val, err := cache.ReadThrough(r.byIDCache, key, func() (Empleado, error) {
		e, err := r.inner.ObtenerPorID(ctx, id)
		if err != nil {
			return Empleado{}, err
		}
		return *e, nil
	})
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func (r *cachedEmpleadoRepository) ListarEmpleados(ctx context.Context, p ListarEmpleadosParams) ([]Empleado, int, error) {
	key := listarEmpleadosKey(p)
	res, err := cache.ReadThrough(r.listCache, key, func() (listarEmpleadosResult, error) {
		items, total, e := r.inner.ListarEmpleados(ctx, p)
		if e != nil {
			return listarEmpleadosResult{}, e
		}
		return listarEmpleadosResult{Items: items, Total: total}, nil
	})
	if err != nil {
		return nil, 0, err
	}
	return res.Items, res.Total, nil
}

// --- Lecturas no cacheadas (volátiles o transversales) ---

func (r *cachedEmpleadoRepository) ExistePorUsuario(ctx context.Context, usuario string) (bool, error) {
	return r.inner.ExistePorUsuario(ctx, usuario)
}

func (r *cachedEmpleadoRepository) ContarAdminsActivosExcluyendo(ctx context.Context, tx *sql.Tx, id uint64) (int, error) {
	return r.inner.ContarAdminsActivosExcluyendo(ctx, tx, id)
}

func (r *cachedEmpleadoRepository) ObtenerTiendaActivaPorID(ctx context.Context, id uint64) error {
	return r.inner.ObtenerTiendaActivaPorID(ctx, id)
}

// --- Escrituras con invalidación ---

func (r *cachedEmpleadoRepository) InsertarEmpleado(ctx context.Context, tx *sql.Tx, emp Empleado, hash string) (uint64, error) {
	id, err := r.inner.InsertarEmpleado(ctx, tx, emp, hash)
	if err != nil {
		return 0, err
	}
	r.listCache.Clear()
	return id, nil
}

func (r *cachedEmpleadoRepository) ActualizarEmpleado(ctx context.Context, tx *sql.Tx, id uint64, emp Empleado) error {
	if err := r.inner.ActualizarEmpleado(ctx, tx, id, emp); err != nil {
		return err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.listCache.Clear()
	return nil
}

func (r *cachedEmpleadoRepository) ActualizarActivo(ctx context.Context, tx *sql.Tx, id uint64, activo bool) error {
	if err := r.inner.ActualizarActivo(ctx, tx, id, activo); err != nil {
		return err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	r.listCache.Clear()
	return nil
}

func (r *cachedEmpleadoRepository) ActualizarContrasena(ctx context.Context, id uint64, hash string) error {
	if err := r.inner.ActualizarContrasena(ctx, id, hash); err != nil {
		return err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	return nil
}

func (r *cachedEmpleadoRepository) MarcarCambioCompletado(ctx context.Context, id uint64) error {
	if err := r.inner.MarcarCambioCompletado(ctx, id); err != nil {
		return err
	}
	r.byIDCache.Delete(fmt.Sprintf("id:%d", id))
	return nil
}

// --- Infraestructura transaccional (passthrough) ---

func (r *cachedEmpleadoRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.inner.BeginTx(ctx)
}

func (r *cachedEmpleadoRepository) RegistrarLog(ctx context.Context, tx *sql.Tx, actorID, empleadoID uint64, accion string, detalle map[string]any) error {
	return r.inner.RegistrarLog(ctx, tx, actorID, empleadoID, accion, detalle)
}

// listarEmpleadosKey construye una clave de caché determinista a partir de los params.
func listarEmpleadosKey(p ListarEmpleadosParams) string {
	tiendaStr := "nil"
	if p.TiendaID != nil {
		tiendaStr = fmt.Sprintf("%d", *p.TiendaID)
	}
	activoStr := "nil"
	if p.Activo != nil {
		activoStr = fmt.Sprintf("%v", *p.Activo)
	}
	return fmt.Sprintf("list:q=%s:tienda=%s:activo=%s:page=%d:limit=%d",
		p.Q, tiendaStr, activoStr, p.Page, p.Limit)
}
