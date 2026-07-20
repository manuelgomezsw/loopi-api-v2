package items_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	it "github.com/manuelgomezsw/loopi-api-v2/internal/items"
)

func newHandler(svc it.Service) *it.Handler {
	return it.NewHandler(svc)
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

// --- TestAccesoSinAdminFallaEnEscritura (quickstart.md §5) ---

func TestAccesoSinAdminFallaEnEscritura(t *testing.T) {
	h := newHandler(&mockSvc{})
	casos := []struct {
		nombre string
		fn     func(w http.ResponseWriter, r *http.Request)
		req    *http.Request
	}{
		{"crear", h.Crear, httptest.NewRequest(http.MethodPost, "/api/v1/items", bytes.NewBufferString(`{}`)).WithContext(ctxRol("barista"))},
		{"editar", h.Editar, newRequestWithID(http.MethodPut, "/api/v1/items/1", "1").WithContext(ctxRol("barista"))},
		{"inactivar", h.Inactivar, newRequestWithID(http.MethodPatch, "/api/v1/items/1/inactivar", "1").WithContext(ctxRol("lider_tienda"))},
		{"reactivar", h.Reactivar, newRequestWithID(http.MethodPatch, "/api/v1/items/1/reactivar", "1").WithContext(ctxRol("lider_tienda"))},
		{"registrar_costo_tienda", h.RegistrarCostoTienda, newRequestWithID(http.MethodPost, "/api/v1/items/1/costos_tienda", "1").WithContext(ctxRol("lider_tienda"))},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.fn(rec, c.req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("esperado 403, obtenido %d", rec.Code)
			}
			body := decodeJSON(t, rec)
			if body["error"] != "sin_permiso" {
				t.Errorf("error esperado 'sin_permiso', obtenido %q", body["error"])
			}
		})
	}
}

func TestAccesoSinTokenFalla(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
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

// --- Lectura del catálogo: cualquier rol autenticado ---

func TestListarPermiteRolNoAdmin(t *testing.T) {
	h := newHandler(&mockSvc{
		listarFunc: func(*it.FiltrosListado) (*it.ListarItemsResponse, error) {
			return &it.ListarItemsResponse{Items: []it.ItemConNombres{}, Total: 0}, nil
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil).WithContext(ctxRol("barista"))
	rec := httptest.NewRecorder()
	h.Listar(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("esperado 200, obtenido %d", rec.Code)
	}
}

// --- Historial de costos por tienda: exclusivo admin, incluso para lectura ---

func TestListarCostosTiendaBloqueaRolNoAdmin(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := newRequestWithID(http.MethodGet, "/api/v1/items/1/costos_tienda", "1").WithContext(ctxRol("lider_tienda"))
	rec := httptest.NewRecorder()
	h.ListarCostosTienda(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("esperado 403, obtenido %d", rec.Code)
	}
}

// --- Body / ID inválidos ---

func TestHandlerCrearBodyInvalidoDevuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", bytes.NewBufferString(`no es json`)).WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.Crear(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}

func TestHandlerObtenerPorIDInvalidoDevuelve400(t *testing.T) {
	h := newHandler(&mockSvc{})
	req := newRequestWithID(http.MethodGet, "/api/v1/items/abc", "abc").WithContext(ctxAdmin())
	rec := httptest.NewRecorder()
	h.ObtenerPorID(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, obtenido %d", rec.Code)
	}
}
