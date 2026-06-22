package cache_test

import (
	"errors"
	"testing"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/cache"
)

func newTestCache(t *testing.T) *cache.EntityCache[string] {
	t.Helper()
	c, err := cache.New[string](time.Minute)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestNewRetornaInstancia(t *testing.T) {
	c, err := cache.New[string](time.Minute)
	if err != nil {
		t.Fatalf("esperaba nil, obtuvo: %v", err)
	}
	if c == nil {
		t.Fatal("esperaba instancia no-nil")
	}
}

func TestGetEnClaveInexistenteRetornaMiss(t *testing.T) {
	c := newTestCache(t)
	_, ok := c.Get("no-existe")
	if ok {
		t.Error("esperaba miss en clave inexistente")
	}
}

func TestSetGetRetornaValorAlmacenado(t *testing.T) {
	c := newTestCache(t)
	c.Set("k", "hola")
	c.Wait()
	val, ok := c.Get("k")
	if !ok {
		t.Fatal("esperaba hit tras Set+Wait")
	}
	if val != "hola" {
		t.Errorf("esperaba 'hola', obtuvo '%s'", val)
	}
}

func TestDeleteDespuesDeSetRetornaMiss(t *testing.T) {
	c := newTestCache(t)
	c.Set("k", "valor")
	c.Wait()
	c.Delete("k")
	c.Wait()
	_, ok := c.Get("k")
	if ok {
		t.Error("esperaba miss tras Delete")
	}
}

func TestClearLimpiaMultiplesClaves(t *testing.T) {
	c := newTestCache(t)
	c.Set("a", "1")
	c.Set("b", "2")
	c.Set("c", "3")
	c.Wait()
	c.Clear()
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.Get(k); ok {
			t.Errorf("clave '%s' debe estar limpia tras Clear", k)
		}
	}
}

func TestReadThroughConHitNoLlamaFetch(t *testing.T) {
	c := newTestCache(t)
	c.Set("k", "cached")
	c.Wait()
	fetchCalls := 0
	val, err := cache.ReadThrough(c, "k", func() (string, error) {
		fetchCalls++
		return "fresh", nil
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if val != "cached" {
		t.Errorf("esperaba 'cached', obtuvo '%s'", val)
	}
	if fetchCalls != 0 {
		t.Errorf("fetch no debió llamarse en hit, llamadas: %d", fetchCalls)
	}
}

func TestReadThroughConMissLlamaFetchYCachea(t *testing.T) {
	c := newTestCache(t)
	fetchCalls := 0
	val, err := cache.ReadThrough(c, "k", func() (string, error) {
		fetchCalls++
		return "fresh", nil
	})
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if val != "fresh" {
		t.Errorf("esperaba 'fresh', obtuvo '%s'", val)
	}
	if fetchCalls != 1 {
		t.Errorf("esperaba 1 llamada a fetch, obtuvo: %d", fetchCalls)
	}
	c.Wait()
	cached, ok := c.Get("k")
	if !ok {
		t.Fatal("esperaba valor en caché tras ReadThrough miss")
	}
	if cached != "fresh" {
		t.Errorf("esperaba 'fresh' en caché, obtuvo '%s'", cached)
	}
}

func TestReadThroughConErrorEnFetchNoCachea(t *testing.T) {
	c := newTestCache(t)
	errBD := errors.New("db error")
	_, err := cache.ReadThrough(c, "k", func() (string, error) {
		return "", errBD
	})
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
	c.Wait()
	_, ok := c.Get("k")
	if ok {
		t.Error("caché no debe modificarse si fetch retorna error")
	}
}
