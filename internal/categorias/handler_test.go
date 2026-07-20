package categorias_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/manuelgomezsw/loopi-api-v2/internal/auth"
	cat "github.com/manuelgomezsw/loopi-api-v2/internal/categorias"
)

func newHandler(svc cat.Service) *cat.Handler {
	return cat.NewHandler(svc)
}

func buildRequest(method, path, body string, adminCtx bool) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if adminCtx {
		r = r.WithContext(ctxAdmin())
	} else {
		r = r.WithContext(ctxRol("barista"))
	}
	return r
}

// --- Crear categoría ---

func TestCrearCategoriaHandler_201(t *testing.T) {
	c := categoriaEjemplo()
	svc := &mockSvc{
		crearCategoriaFunc: func(nombre string, userID uint64, rol string) (*cat.CategoriaResponse, error) {
			return &cat.CategoriaResponse{Categoria: *c, Subcategorias: []cat.SubcategoriaResponse{}}, nil
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/categorias", `{"nombre":"Lácteo"}`, true)
	h.CrearCategoria(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("esperaba 201, obtuvo %d", w.Code)
	}
}

func TestCrearCategoriaHandler_400_NombreVacio(t *testing.T) {
	svc := &mockSvc{}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/categorias", `{"nombre":""}`, true)
	h.CrearCategoria(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperaba 400, obtuvo %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "nombre_requerido" {
		t.Errorf("error esperado 'nombre_requerido', obtuvo %q", body["error"])
	}
}

func TestCrearCategoriaHandler_401_SinToken(t *testing.T) {
	svc := &mockSvc{}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/categorias", strings.NewReader(`{"nombre":"X"}`))
	r = r.WithContext(auth.ContextWithClaims(r.Context(), nil))
	h.CrearCategoria(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("esperaba 401, obtuvo %d", w.Code)
	}
}

func TestCrearCategoriaHandler_403_RolNoAdmin(t *testing.T) {
	svc := &mockSvc{}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/categorias", `{"nombre":"X"}`, false)
	h.CrearCategoria(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("esperaba 403, obtuvo %d", w.Code)
	}
}

func TestCrearCategoriaHandler_409_Duplicado(t *testing.T) {
	svc := &mockSvc{
		crearCategoriaFunc: func(nombre string, userID uint64, rol string) (*cat.CategoriaResponse, error) {
			return nil, cat.ErrNombreDuplicado
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/categorias", `{"nombre":"Lácteo"}`, true)
	h.CrearCategoria(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("esperaba 409, obtuvo %d", w.Code)
	}
}

// --- Crear subcategoría ---

func TestCrearSubcategoriaHandler_201(t *testing.T) {
	sub := subcategoriaEjemplo()
	svc := &mockSvc{
		crearSubcategoriaFunc: func(nombre string, categoriaID, userID uint64, rol string) (*cat.SubcategoriaResponse, error) {
			return &cat.SubcategoriaResponse{Subcategoria: *sub, TotalItems: 0}, nil
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/subcategorias", `{"nombre":"Quesos","categoria_id":1}`, true)
	h.CrearSubcategoria(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("esperaba 201, obtuvo %d", w.Code)
	}
}

func TestCrearSubcategoriaHandler_400_SinCategoria(t *testing.T) {
	svc := &mockSvc{}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/subcategorias", `{"nombre":"Quesos","categoria_id":0}`, true)
	h.CrearSubcategoria(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperaba 400, obtuvo %d", w.Code)
	}
}

func TestCrearSubcategoriaHandler_403_NoAdmin(t *testing.T) {
	svc := &mockSvc{}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/subcategorias", `{"nombre":"X","categoria_id":1}`, false)
	h.CrearSubcategoria(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("esperaba 403, obtuvo %d", w.Code)
	}
}

func TestCrearSubcategoriaHandler_404_CatNoExiste(t *testing.T) {
	svc := &mockSvc{
		crearSubcategoriaFunc: func(nombre string, categoriaID, userID uint64, rol string) (*cat.SubcategoriaResponse, error) {
			return nil, cat.ErrCategoriaNoEncontrada
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/subcategorias", `{"nombre":"X","categoria_id":99}`, true)
	h.CrearSubcategoria(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("esperaba 404, obtuvo %d", w.Code)
	}
}

func TestCrearSubcategoriaHandler_409_Duplicado(t *testing.T) {
	svc := &mockSvc{
		crearSubcategoriaFunc: func(nombre string, categoriaID, userID uint64, rol string) (*cat.SubcategoriaResponse, error) {
			return nil, cat.ErrNombreDuplicado
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/subcategorias", `{"nombre":"quesos","categoria_id":1}`, true)
	h.CrearSubcategoria(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("esperaba 409, obtuvo %d", w.Code)
	}
}

func TestCrearSubcategoriaHandler_422_CatPadreInactiva(t *testing.T) {
	svc := &mockSvc{
		crearSubcategoriaFunc: func(nombre string, categoriaID, userID uint64, rol string) (*cat.SubcategoriaResponse, error) {
			return nil, cat.ErrCategoriaPadreInactiva
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("POST", "/api/v1/subcategorias", `{"nombre":"X","categoria_id":1}`, true)
	h.CrearSubcategoria(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperaba 422, obtuvo %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "categoria_padre_inactiva" {
		t.Errorf("error esperado 'categoria_padre_inactiva', obtuvo %q", body["error"])
	}
}

// --- Inactivar categoría ---

func TestInactivarCategoriaHandler_422_YaInactiva(t *testing.T) {
	svc := &mockSvc{
		inactivarCategoriaFunc: func(id, userID uint64, rol string) (*cat.InactivarCategoriaResponse, error) {
			return nil, cat.ErrCategoriaYaInactiva
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("PATCH", "/api/v1/categorias/1/inactivar", "", true)
	r.SetPathValue("id", "1")
	h.InactivarCategoria(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperaba 422, obtuvo %d", w.Code)
	}
}

// --- Reactivar subcategoría ---

func TestReactivarSubcategoriaHandler_422_CatPadreInactiva(t *testing.T) {
	svc := &mockSvc{
		reactivarSubcategoriaFunc: func(id, userID uint64, rol string) (*cat.Subcategoria, error) {
			return nil, cat.ErrCategoriaPadreInactiva
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("PATCH", "/api/v1/subcategorias/1/reactivar", "", true)
	r.SetPathValue("id", "1")
	h.ReactivarSubcategoria(w, r)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("esperaba 422, obtuvo %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "categoria_padre_inactiva" {
		t.Errorf("error esperado 'categoria_padre_inactiva', obtuvo %q", body["error"])
	}
}

// --- Listar catálogo ---

func TestListarCatalogoHandler_400_EstadoInvalido(t *testing.T) {
	svc := &mockSvc{}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("GET", "/api/v1/categorias?estado=invalido", "", true)
	h.ListarCatalogo(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperaba 400, obtuvo %d", w.Code)
	}
	var body map[string]string
	_ = json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "estado_invalido" {
		t.Errorf("error esperado 'estado_invalido', obtuvo %q", body["error"])
	}
}

func TestListarCatalogoHandler_200(t *testing.T) {
	svc := &mockSvc{
		obtenerCatalogoFunc: func(estado string) (*cat.CatalogoResponse, error) {
			return &cat.CatalogoResponse{Categorias: []cat.CategoriaResponse{}, Total: 0}, nil
		},
	}
	h := newHandler(svc)
	w := httptest.NewRecorder()
	r := buildRequest("GET", "/api/v1/categorias", "", true)
	h.ListarCatalogo(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("esperaba 200, obtuvo %d", w.Code)
	}
}
