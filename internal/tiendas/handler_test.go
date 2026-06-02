package tiendas_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/internal/tiendas"
)

// newHandlerTest crea un TiendaHandler con el mock de servicio dado.
func newHandlerTest(svc tiendas.TiendaService) *tiendas.TiendaHandler {
	return tiendas.NewTiendaHandler(svc)
}

// decodeJSON decodifica el body de la respuesta en un map genérico.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("error decodificando respuesta JSON: %v", err)
	}
	return m
}

// --- Listar ---

func TestHandler_Listar_SinJWT_Devuelve401(t *testing.T) {
	h := newHandlerTest(&mockService{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas", nil)
	rec := httptest.NewRecorder()

	h.Listar(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestHandler_Listar_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandlerTest(&mockService{
		listarFunc: func(string, int, int) (tiendas.ListaTiendasResponse, error) {
			return tiendas.ListaTiendasResponse{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas", nil).WithContext(ctxRol("lider_tienda"))
	rec := httptest.NewRecorder()

	h.Listar(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["error"] != "sin_permiso" {
		t.Errorf("error esperado 'sin_permiso', obtenido %q", body["error"])
	}
}

func TestHandler_Listar_EstadoInvalido_Devuelve400(t *testing.T) {
	h := newHandlerTest(&mockService{
		listarFunc: func(string, int, int) (tiendas.ListaTiendasResponse, error) {
			return tiendas.ListaTiendasResponse{}, &tiendas.ValidationError{
				Codigo: "estado_invalido", Mensaje: "estado no válido", Campo: "estado",
			}
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas?estado=malo", nil).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Listar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_Listar_OK_Devuelve200ConDatos(t *testing.T) {
	h := newHandlerTest(&mockService{
		listarFunc: func(string, int, int) (tiendas.ListaTiendasResponse, error) {
			return tiendas.ListaTiendasResponse{
				Datos:  []tiendas.TiendaResponse{{ID: 1, Codigo: "TDA-001"}},
				Total:  1,
				Pagina: 1,
				Limite: 50,
			}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas", nil).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Listar(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["total"].(float64) != 1 {
		t.Errorf("total esperado 1, obtenido %v", body["total"])
	}
}

func TestHandler_Listar_ErrorInterno_Devuelve500(t *testing.T) {
	h := newHandlerTest(&mockService{
		listarFunc: func(string, int, int) (tiendas.ListaTiendasResponse, error) {
			return tiendas.ListaTiendasResponse{}, errors.New("bd caída")
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas", nil).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Listar(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("esperado 500, obtenido %d", rec.Code)
	}
}

// --- Crear ---

func TestHandler_Crear_SinJWT_Devuelve401(t *testing.T) {
	h := newHandlerTest(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	h.Crear(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandlerTest(&mockService{
		crearFunc: func(tiendas.TiendaRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	body := `{"codigo":"X","nombre":"N","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas", strings.NewReader(body)).
		WithContext(ctxRol("barista"))
	rec := httptest.NewRecorder()

	h.Crear(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_CampoFaltante_Devuelve400ConCampo(t *testing.T) {
	h := newHandlerTest(&mockService{
		crearFunc: func(tiendas.TiendaRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	// Falta "nombre"
	body := `{"codigo":"X","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas", strings.NewReader(body)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Crear(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["campo"] != "nombre" {
		t.Errorf("campo esperado 'nombre', obtenido %q", resp["campo"])
	}
}

func TestHandler_Crear_NombreDuplicado_Devuelve409(t *testing.T) {
	h := newHandlerTest(&mockService{
		crearFunc: func(tiendas.TiendaRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrNombreDuplicado
		},
	})
	body := `{"codigo":"X","nombre":"Existe","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas", strings.NewReader(body)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Crear(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["error"] != "nombre_duplicado" {
		t.Errorf("error esperado 'nombre_duplicado', obtenido %q", resp["error"])
	}
	if resp["campo"] != "nombre" {
		t.Errorf("campo esperado 'nombre', obtenido %q", resp["campo"])
	}
}

func TestHandler_Crear_CodigoDuplicado_Devuelve409(t *testing.T) {
	h := newHandlerTest(&mockService{
		crearFunc: func(tiendas.TiendaRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrCodigoDuplicado
		},
	})
	body := `{"codigo":"DUP","nombre":"N","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas", strings.NewReader(body)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Crear(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["error"] != "codigo_duplicado" {
		t.Errorf("error esperado 'codigo_duplicado', obtenido %q", resp["error"])
	}
}

func TestHandler_Crear_OK_Devuelve201(t *testing.T) {
	h := newHandlerTest(&mockService{
		crearFunc: func(req tiendas.TiendaRequest, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{ID: 1, Codigo: req.Codigo, Nombre: req.Nombre, Activo: true}, nil
		},
	})
	body := `{"codigo":"TDA-001","nombre":"Norte","direccion":"Calle 1","ciudad":"Bogotá","telefono":"300"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas", strings.NewReader(body)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Crear(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("esperado 201, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["activo"] != true {
		t.Errorf("activo esperado true, obtenido %v", resp["activo"])
	}
}

// --- ObtenerPorID ---

func TestHandler_ObtenerPorID_IDInvalido_Devuelve400(t *testing.T) {
	h := newHandlerTest(&mockService{
		obtenerFunc: func(id uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas/abc", nil).WithContext(ctxAdmin())
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()

	h.ObtenerPorID(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_ObtenerPorID_NoExiste_Devuelve404(t *testing.T) {
	h := newHandlerTest(&mockService{
		obtenerFunc: func(id uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrTiendaNoEncontrada
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas/999", nil).WithContext(ctxAdmin())
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()

	h.ObtenerPorID(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obtenido %d", rec.Code)
	}
}

func TestHandler_ObtenerPorID_OK_Devuelve200(t *testing.T) {
	h := newHandlerTest(&mockService{
		obtenerFunc: func(id uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{ID: id, Codigo: "TDA-001"}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas/1", nil).WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.ObtenerPorID(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Actualizar ---

func TestHandler_Actualizar_CampoFaltante_Devuelve400(t *testing.T) {
	h := newHandlerTest(&mockService{
		actualizarFunc: func(uint64, tiendas.TiendaUpdateRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	body := `{"nombre":"N","direccion":"D","ciudad":"C"}` // falta telefono
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tiendas/1", strings.NewReader(body)).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Actualizar(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_Actualizar_NombreDuplicado_Devuelve409(t *testing.T) {
	h := newHandlerTest(&mockService{
		actualizarFunc: func(uint64, tiendas.TiendaUpdateRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrNombreDuplicado
		},
	})
	body := `{"nombre":"Duplicado","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tiendas/1", strings.NewReader(body)).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Actualizar(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Actualizar_OK_Devuelve200(t *testing.T) {
	h := newHandlerTest(&mockService{
		actualizarFunc: func(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{ID: id, Nombre: req.Nombre, Codigo: "TDA-001"}, nil
		},
	})
	body := `{"nombre":"Nuevo","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tiendas/1", strings.NewReader(body)).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Actualizar(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Inactivar ---

func TestHandler_Inactivar_YaInactiva_Devuelve422(t *testing.T) {
	h := newHandlerTest(&mockService{
		inactivarFunc: func(id, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrTiendaYaInactiva
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/1/inactivar", nil).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Inactivar(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperado 422, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["error"] != "tienda_ya_inactiva" {
		t.Errorf("error esperado 'tienda_ya_inactiva', obtenido %q", resp["error"])
	}
}

func TestHandler_Inactivar_NoExiste_Devuelve404(t *testing.T) {
	h := newHandlerTest(&mockService{
		inactivarFunc: func(id, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrTiendaNoEncontrada
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/999/inactivar", nil).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "999")
	rec := httptest.NewRecorder()

	h.Inactivar(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obtenido %d", rec.Code)
	}
}

func TestHandler_Inactivar_OK_Devuelve200ConActivoFalse(t *testing.T) {
	h := newHandlerTest(&mockService{
		inactivarFunc: func(id, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{ID: id, Activo: false}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/1/inactivar", nil).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Inactivar(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["activo"] != false {
		t.Errorf("activo debe ser false, obtenido %v", resp["activo"])
	}
}

// --- Reactivar ---

func TestHandler_Reactivar_YaActiva_Devuelve422(t *testing.T) {
	h := newHandlerTest(&mockService{
		reactivarFunc: func(id, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, tiendas.ErrTiendaYaActiva
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/1/reactivar", nil).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Reactivar(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperado 422, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["error"] != "tienda_ya_activa" {
		t.Errorf("error esperado 'tienda_ya_activa', obtenido %q", resp["error"])
	}
}

func TestHandler_Reactivar_OK_Devuelve200ConActivoTrue(t *testing.T) {
	h := newHandlerTest(&mockService{
		reactivarFunc: func(id, adminID uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{ID: id, Activo: true}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/1/reactivar", nil).
		WithContext(ctxAdmin())
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Reactivar(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
	resp := decodeJSON(t, rec)
	if resp["activo"] != true {
		t.Errorf("activo debe ser true, obtenido %v", resp["activo"])
	}
}

// --- ObtenerPorID / Actualizar / Inactivar / Reactivar — 403 ---

func TestHandler_ObtenerPorID_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandlerTest(&mockService{
		obtenerFunc: func(uint64) (tiendas.TiendaResponse, error) { return tiendas.TiendaResponse{}, nil },
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas/1", nil).WithContext(ctxRol("lider_tienda"))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.ObtenerPorID(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

func TestHandler_Actualizar_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandlerTest(&mockService{
		actualizarFunc: func(uint64, tiendas.TiendaUpdateRequest, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	body := `{"nombre":"N","direccion":"D","ciudad":"C","telefono":"T"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tiendas/1", strings.NewReader(body)).
		WithContext(ctxRol("lider_compras"))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Actualizar(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

func TestHandler_Inactivar_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandlerTest(&mockService{
		inactivarFunc: func(uint64, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/1/inactivar", nil).
		WithContext(ctxRol("barista"))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Inactivar(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

func TestHandler_Reactivar_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandlerTest(&mockService{
		reactivarFunc: func(uint64, uint64) (tiendas.TiendaResponse, error) {
			return tiendas.TiendaResponse{}, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tiendas/1/reactivar", nil).
		WithContext(ctxRol("lider_tienda"))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	h.Reactivar(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

// Cubre parseIntQuery con valores explícitos válidos.
func TestHandler_Listar_ConPaginaYLimiteExplicitos_OK(t *testing.T) {
	var paginaRecibida, limiteRecibido int
	h := newHandlerTest(&mockService{
		listarFunc: func(estado string, pagina, limite int) (tiendas.ListaTiendasResponse, error) {
			paginaRecibida = pagina
			limiteRecibido = limite
			return tiendas.ListaTiendasResponse{Datos: []tiendas.TiendaResponse{}, Total: 0, Pagina: pagina, Limite: limite}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tiendas?pagina=2&limite=10", nil).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()

	h.Listar(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
	if paginaRecibida != 2 {
		t.Errorf("pagina esperada 2, obtenida %d", paginaRecibida)
	}
	if limiteRecibido != 10 {
		t.Errorf("limite esperado 10, obtenido %d", limiteRecibido)
	}
}

// --- NewTiendaHandler ---

func TestNewTiendaHandler_NoDevuelveNil(t *testing.T) {
	svc := &mockService{}
	h := tiendas.NewTiendaHandler(svc)
	if h == nil {
		t.Error("NewTiendaHandler no debe devolver nil")
	}
}
