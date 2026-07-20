package proveedores_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pv "github.com/manuelgomezsw/loopi-api-v2/internal/proveedores"
)

func newHandler(svc pv.Service) *pv.Handler {
	return pv.NewHandler(svc)
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("error decodificando JSON: %v", err)
	}
	return m
}

func newRequestWithID(method, url, id string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	req.SetPathValue("id", id)
	return req
}

// --- Acceso transversal (401/403 en todos los endpoints) ---

func TestAccesoSinTokenFalla(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proveedores", nil)
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["error"] != "no_autenticado" {
		t.Errorf("error esperado 'no_autenticado', obtenido %q", body["error"])
	}
}

func TestAccesoLiderTiendaFalla(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proveedores", nil).WithContext(ctxRol("lider_tienda"))
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["error"] != "acceso_denegado" {
		t.Errorf("error esperado 'acceso_denegado', obtenido %q", body["error"])
	}
}

// --- Crear ---

func TestHandler_Crear_BodyInvalido_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proveedores",
		bytes.NewBufferString(`no es json`)).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_CampoFaltante_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{
		crearFunc: func(*pv.CrearProveedorRequest, uint64, string) (*pv.Proveedor, error) {
			return nil, &pv.ValidationError{Codigo: "campo_requerido", Mensaje: "La razón social es obligatoria.", Campo: "razon_social"}
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proveedores",
		bytes.NewBufferString(`{"nit":"PROV-002"}`)).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["campo"] != "razon_social" {
		t.Errorf("campo esperado 'razon_social', obtenido %q", body["campo"])
	}
}

func TestHandler_Crear_NITDuplicado_Devuelve409(t *testing.T) {
	h := newHandler(&mockSvc{
		crearFunc: func(*pv.CrearProveedorRequest, uint64, string) (*pv.Proveedor, error) {
			return nil, pv.ErrNITDuplicado
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proveedores",
		bytes.NewBufferString(`{"razon_social":"Otro","nit":"900123456-7"}`)).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_Exitoso_Devuelve201(t *testing.T) {
	h := newHandler(&mockSvc{
		crearFunc: func(*pv.CrearProveedorRequest, uint64, string) (*pv.Proveedor, error) {
			return proveedorEjemplo(), nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proveedores",
		bytes.NewBufferString(`{"razon_social":"Distribuidora La Cosecha S.A.S","nit":"900123456-7"}`)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("esperado 201, obtenido %d", rec.Code)
	}
}

// --- Listar ---

func TestHandler_Listar_EstadoInvalido_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proveedores?estado=bogus", nil).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["error"] != "estado_invalido" {
		t.Errorf("error esperado 'estado_invalido', obtenido %q", body["error"])
	}
}

func TestHandler_Listar_Exitoso_Devuelve200(t *testing.T) {
	h := newHandler(&mockSvc{
		listarFunc: func(*pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
			return &pv.ListarProveedoresResponse{Proveedores: []pv.Proveedor{}, Total: 0, Page: 1, Limit: 50}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proveedores?estado=activo", nil).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- ObtenerPorID ---

func TestHandler_ObtenerPorID_NoEncontrado_Devuelve404(t *testing.T) {
	h := newHandler(&mockSvc{
		obtenerPorIDFunc: func(uint64) (*pv.ProveedorDetalleResponse, error) {
			return nil, pv.ErrProveedorNoEncontrado
		},
	})
	req := newRequestWithID(http.MethodGet, "/api/v1/proveedores/999", "999").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.ObtenerPorID(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obtenido %d", rec.Code)
	}
}

func TestHandler_ObtenerPorID_Exitoso_Devuelve200(t *testing.T) {
	h := newHandler(&mockSvc{
		obtenerPorIDFunc: func(uint64) (*pv.ProveedorDetalleResponse, error) {
			return &pv.ProveedorDetalleResponse{Proveedor: *proveedorEjemplo(), ItemsAsignados: 3}, nil
		},
	})
	req := newRequestWithID(http.MethodGet, "/api/v1/proveedores/1", "1").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.ObtenerPorID(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Editar ---

func newRequestWithIDAndBody(method, url, id, body string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewBufferString(body))
	req.SetPathValue("id", id)
	return req
}

func TestHandler_Editar_NITDuplicado_Devuelve409(t *testing.T) {
	h := newHandler(&mockSvc{
		editarFunc: func(uint64, *pv.EditarProveedorRequest, uint64, string) (*pv.Proveedor, error) {
			return nil, pv.ErrNITDuplicado
		},
	})
	req := newRequestWithIDAndBody(http.MethodPut, "/api/v1/proveedores/1", "1", `{"nit":"PROV-001"}`).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Editar(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Editar_Exitoso_Devuelve200(t *testing.T) {
	h := newHandler(&mockSvc{
		editarFunc: func(uint64, *pv.EditarProveedorRequest, uint64, string) (*pv.Proveedor, error) {
			return proveedorEjemplo(), nil
		},
	})
	req := newRequestWithIDAndBody(http.MethodPut, "/api/v1/proveedores/1", "1", `{"nombre_contacto":"María López"}`).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Editar(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Inactivar ---

func TestHandler_Inactivar_YaInactivo_Devuelve409(t *testing.T) {
	h := newHandler(&mockSvc{
		inactivarFunc: func(uint64, uint64, string) (*pv.CambiarEstadoResponse, error) {
			return nil, pv.ErrYaInactivo
		},
	})
	req := newRequestWithID(http.MethodPatch, "/api/v1/proveedores/1/inactivar", "1").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Inactivar(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Inactivar_Exitoso_Devuelve200(t *testing.T) {
	h := newHandler(&mockSvc{
		inactivarFunc: func(uint64, uint64, string) (*pv.CambiarEstadoResponse, error) {
			return &pv.CambiarEstadoResponse{ID: 1, Activo: false, Mensaje: "ok"}, nil
		},
	})
	req := newRequestWithID(http.MethodPatch, "/api/v1/proveedores/1/inactivar", "1").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Inactivar(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Activar ---

func TestHandler_Activar_YaActivo_Devuelve409(t *testing.T) {
	h := newHandler(&mockSvc{
		activarFunc: func(uint64, uint64, string) (*pv.CambiarEstadoResponse, error) {
			return nil, pv.ErrYaActivo
		},
	})
	req := newRequestWithID(http.MethodPatch, "/api/v1/proveedores/1/activar", "1").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Activar(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Activar_Exitoso_Devuelve200(t *testing.T) {
	h := newHandler(&mockSvc{
		activarFunc: func(uint64, uint64, string) (*pv.CambiarEstadoResponse, error) {
			return &pv.CambiarEstadoResponse{ID: 1, Activo: true, Mensaje: "ok"}, nil
		},
	})
	req := newRequestWithID(http.MethodPatch, "/api/v1/proveedores/1/activar", "1").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Activar(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}
