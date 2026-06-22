package unidades_medida_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	um "github.com/manuelgomezsw/loopi-api-v2/internal/unidades_medida"
)

func newHandler(svc um.UMService) *um.UMHandler {
	return um.NewHandler(svc)
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("error decodificando JSON: %v", err)
	}
	return m
}

// --- Crear ---

func TestHandler_Crear_SinJWT_Devuelve401(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unidades_medida", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unidades_medida",
		bytes.NewBufferString(`{"codigo":"oz","nombre":"Onza","tipo_medida":"peso","factor_conversion":28}`)).
		WithContext(ctxRol("lider_tienda"))
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
	body := decodeJSON(t, rec)
	if body["error"] != "acceso_denegado" {
		t.Errorf("error esperado 'acceso_denegado', obtenido %q", body["error"])
	}
}

func TestHandler_Crear_BodyInvalido_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unidades_medida",
		bytes.NewBufferString(`no es json`)).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_CampoFaltante_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unidades_medida",
		bytes.NewBufferString(`{"nombre":"Onza","factor_conversion":28}`)).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_CodigoDuplicado_Devuelve409(t *testing.T) {
	h := newHandler(&mockSvc{
		crearFunc: func(*um.CrearUMRequest, uint64, string) (*um.UnidadMedida, error) {
			return nil, um.ErrCodigoDuplicado
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unidades_medida",
		bytes.NewBufferString(`{"codigo":"kg","nombre":"Kilo","tipo_medida":"peso","factor_conversion":1000}`)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Crear_Exitoso_Devuelve201(t *testing.T) {
	h := newHandler(&mockSvc{
		crearFunc: func(*um.CrearUMRequest, uint64, string) (*um.UnidadMedida, error) {
			return umEjemplo(), nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unidades_medida",
		bytes.NewBufferString(`{"codigo":"kg","nombre":"Kilogramo","tipo_medida":"peso","factor_conversion":1000}`)).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("esperado 201, obtenido %d", rec.Code)
	}
}

// --- Listar ---

func TestHandler_Listar_SinJWT_Devuelve401(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unidades_medida", nil)
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("esperado 401, obtenido %d", rec.Code)
	}
}

func TestHandler_Listar_TipoInvalido_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unidades_medida?tipo=area", nil).
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandler_Listar_Exitoso_Devuelve200(t *testing.T) {
	h := newHandler(&mockSvc{
		listarFunc: func(*um.ListarUMParams) (*um.ListarUMResponse, error) {
			return &um.ListarUMResponse{UnidadesMedida: []um.UnidadMedida{}, Total: 0, Page: 1, Limit: 50}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/unidades_medida", nil).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Inactivar ---

func TestHandler_Inactivar_RolNoAdmin_Devuelve403(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := newRequestWithID(http.MethodPatch, "/api/v1/unidades_medida/4/inactivar", "4").
		WithContext(ctxRol("barista"))
	rec := httptest.NewRecorder()
	h.Inactivar(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

func TestHandler_Inactivar_YaInactiva_Devuelve409(t *testing.T) {
	h := newHandler(&mockSvc{
		inactivarFunc: func(uint64, uint64, string) (*um.InactivarResponse, error) {
			return nil, um.ErrYaInactiva
		},
	})
	req := newRequestWithID(http.MethodPatch, "/api/v1/unidades_medida/4/inactivar", "4").
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Inactivar(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("esperado 409, obtenido %d", rec.Code)
	}
}

func TestHandler_Inactivar_UnidadBase_Devuelve422(t *testing.T) {
	h := newHandler(&mockSvc{
		inactivarFunc: func(uint64, uint64, string) (*um.InactivarResponse, error) {
			return nil, um.ErrUnidadBaseNoInactivable
		},
	})
	req := newRequestWithID(http.MethodPatch, "/api/v1/unidades_medida/1/inactivar", "1").
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Inactivar(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperado 422, obtenido %d", rec.Code)
	}
}

// --- ObtenerImpacto ---

func TestHandler_ObtenerImpacto_NoEncontrada_Devuelve404(t *testing.T) {
	h := newHandler(&mockSvc{
		obtenerImpactoFunc: func(uint64) (*um.ImpactoResponse, error) {
			return nil, um.ErrUnidadNoEncontrada
		},
	})
	req := newRequestWithID(http.MethodGet, "/api/v1/unidades_medida/999/impacto", "999").
		WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.ObtenerImpacto(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("esperado 404, obtenido %d", rec.Code)
	}
}

// --- Editar ---

func TestHandler_Editar_SinCampos_Devuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := newRequestWithID(http.MethodPut, "/api/v1/unidades_medida/4", "4")
	req = req.WithContext(ctxAdmin())
	req.Body = http.NoBody
	rec := httptest.NewRecorder()
	h.Editar(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

// newRequestWithID crea un Request con PathValue "id" seteado.
func newRequestWithID(method, url, id string) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	req.SetPathValue("id", id)
	return req
}
