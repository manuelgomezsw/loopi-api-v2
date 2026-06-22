package cache

import (
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

// EntityCache es una caché tipada para una entidad de dominio.
// Cada entidad crea su propia instancia; Clear() afecta solo esa entidad.
type EntityCache[T any] struct {
	c   *ristretto.Cache[string, T]
	ttl time.Duration
}

// New crea una EntityCache con el TTL indicado.
// NumCounters y MaxCost son adecuados para catálogos de ≤ 10 000 entradas.
func New[T any](ttl time.Duration) (*EntityCache[T], error) {
	c, err := ristretto.NewCache(&ristretto.Config[string, T]{
		NumCounters: 1e4,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("cache.New: %w", err)
	}
	return &EntityCache[T]{c: c, ttl: ttl}, nil
}

// Get retorna (value, true) si la clave existe, o (zero, false) en miss.
func (e *EntityCache[T]) Get(key string) (T, bool) {
	return e.c.Get(key)
}

// Set almacena value con el TTL configurado.
func (e *EntityCache[T]) Set(key string, value T) {
	e.c.SetWithTTL(key, value, 1, e.ttl)
}

// Delete elimina una clave individual.
func (e *EntityCache[T]) Delete(key string) {
	e.c.Del(key)
}

// Clear invalida todas las entradas de esta caché.
// Llamar en cualquier operación de escritura sobre la entidad.
func (e *EntityCache[T]) Clear() {
	e.c.Clear()
}

// Wait espera a que el buffer asíncrono de Ristretto procese los Set pendientes.
// Solo necesario en tests para garantizar consistencia tras un Set.
func (e *EntityCache[T]) Wait() {
	e.c.Wait()
}

// ReadThrough es el helper canónico para métodos de lectura del decorador.
// En hit retorna el valor cacheado sin llamar fetch.
// En miss llama fetch, cachea el resultado y lo retorna.
// Si fetch retorna error, no modifica la caché.
func ReadThrough[T any](c *EntityCache[T], key string, fetch func() (T, error)) (T, error) {
	if val, ok := c.Get(key); ok {
		return val, nil
	}
	val, err := fetch()
	if err != nil {
		return val, err
	}
	c.Set(key, val)
	return val, nil
}
