package unidades_medida

import (
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

const (
	cacheTTL  = 5 * time.Minute
	keyAll    = "um:all"
	keyByID   = "um:id:%d"
	keyByTipo = "um:tipo:%s"
)

// umCache envuelve una instancia de Ristretto para el catálogo de unidades de medida.
type umCache struct {
	r *ristretto.Cache[string, any]
}

func newUMCache() (*umCache, error) {
	r, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 100_000,
		MaxCost:     1 << 20, // 1 MB
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("inicializar ristretto cache: %w", err)
	}
	return &umCache{r: r}, nil
}

func (c *umCache) get(key string) (any, bool) {
	return c.r.Get(key)
}

func (c *umCache) set(key string, val any) {
	c.r.SetWithTTL(key, val, 1, cacheTTL)
}

// invalidarCatalogo borra todas las claves del módulo tras cualquier operación de escritura.
func (c *umCache) invalidarCatalogo(id uint64) {
	c.r.Del(keyAll)
	c.r.Del(fmt.Sprintf(keyByID, id))
	c.r.Del(fmt.Sprintf(keyByTipo, "peso"))
	c.r.Del(fmt.Sprintf(keyByTipo, "volumen"))
	c.r.Del(fmt.Sprintf(keyByTipo, "unidad"))
}
