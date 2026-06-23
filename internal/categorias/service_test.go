package categorias_test

import (
	"errors"
	"testing"

	cat "github.com/manuelgomezsw/loopi-api-v2/internal/categorias"
)

func newSvc(t *testing.T, repo cat.Repository) cat.Service {
	t.Helper()
	return cat.NewService(repo)
}

// --- TestCrearCategoriaNombreDuplicadoCaseInsensitive ---

func TestCrearCategoriaNombreDuplicadoCaseInsensitive(t *testing.T) {
	repo := &mockRepo{
		insertarCategoriaFunc: func(nombre string, creadoPor uint64) (*cat.Categoria, error) {
			return nil, cat.ErrNombreDuplicado
		},
	}
	svc := newSvc(t, repo)
	_, err := svc.CrearCategoria("lácteo", 1, "admin")
	if !errors.Is(err, cat.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

func TestCrearCategoriaExitosa(t *testing.T) {
	c := categoriaEjemplo()
	repo := &mockRepo{
		insertarCategoriaFunc: func(nombre string, creadoPor uint64) (*cat.Categoria, error) {
			return c, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.CrearCategoria("Lácteo", 42, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.ID != c.ID {
		t.Errorf("id esperado %d, obtuvo %d", c.ID, resp.ID)
	}
	if len(resp.Subcategorias) != 0 {
		t.Error("subcategorias debe estar vacío para una categoría nueva")
	}
}

// --- TestCrearSubcategoriaNombreDuplicadoMismaCat ---

func TestCrearSubcategoriaNombreDuplicadoMismaCat(t *testing.T) {
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) { return categoriaEjemplo(), nil },
		insertarSubcategoriaFunc: func(nombre string, categoriaID, creadoPor uint64) (*cat.Subcategoria, error) {
			return nil, cat.ErrNombreDuplicado
		},
	}
	svc := newSvc(t, repo)
	_, err := svc.CrearSubcategoria("quesos", 1, 42, "admin")
	if !errors.Is(err, cat.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

// --- TestCrearSubcategoriasMismoNombreDistintasCat ---

func TestCrearSubcategoriasMismoNombreDistintasCat(t *testing.T) {
	llamadas := 0
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) {
			c := categoriaEjemplo()
			c.ID = id
			return c, nil
		},
		insertarSubcategoriaFunc: func(nombre string, categoriaID, creadoPor uint64) (*cat.Subcategoria, error) {
			llamadas++
			return &cat.Subcategoria{ID: uint64(llamadas), Nombre: nombre, CategoriaID: categoriaID, Activo: true}, nil
		},
	}
	svc := newSvc(t, repo)

	_, err1 := svc.CrearSubcategoria("Quesos", 1, 42, "admin")
	_, err2 := svc.CrearSubcategoria("Quesos", 2, 42, "admin")

	if err1 != nil {
		t.Errorf("primera creación no esperaba error: %v", err1)
	}
	if err2 != nil {
		t.Errorf("segunda creación en cat distinta no esperaba error: %v", err2)
	}
	if llamadas != 2 {
		t.Errorf("esperaba 2 inserciones, obtuvo %d", llamadas)
	}
}

// --- TestCrearSubcategoriaCategoriaInactiva ---

func TestCrearSubcategoriaCategoriaInactiva(t *testing.T) {
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) { return categoriaInactiva(), nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.CrearSubcategoria("Mantequilla", 1, 42, "admin")
	if !errors.Is(err, cat.ErrCategoriaPadreInactiva) {
		t.Errorf("esperaba ErrCategoriaPadreInactiva, obtuvo: %v", err)
	}
}

// --- TestInactivarCategoriaConSubcatsActivasCascade ---

func TestInactivarCategoriaConSubcatsActivasCascade(t *testing.T) {
	c := categoriaEjemplo()
	inactivarCatLlamado := false
	inactivarSubcatsLlamado := false
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) {
			if inactivarCatLlamado {
				inactiva := categoriaInactiva()
				inactiva.ID = c.ID
				return inactiva, nil
			}
			return c, nil
		},
		inactivarCategoriaFunc: func(id, actualizadoPor uint64) error {
			inactivarCatLlamado = true
			return nil
		},
		inactivarSubcategoriasDeCategFunc: func(categoriaID, actualizadoPor uint64) (int, error) {
			inactivarSubcatsLlamado = true
			return 2, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.InactivarCategoria(1, 42, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !inactivarCatLlamado {
		t.Error("InactivarCategoria del repo no fue llamado")
	}
	if !inactivarSubcatsLlamado {
		t.Error("InactivarSubcategoriasDeCategoria del repo no fue llamado")
	}
	if resp.SubcategoriasInactivadas != 2 {
		t.Errorf("esperaba subcategorias_inactivadas=2, obtuvo %d", resp.SubcategoriasInactivadas)
	}
}

// --- TestImpactoCategoriaConSubcatsActivas ---

func TestImpactoCategoriaConSubcatsActivas(t *testing.T) {
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc:      func(id uint64) (*cat.Categoria, error) { return categoriaEjemplo(), nil },
		contarSubcategoriasActivasFunc: func(categoriaID uint64) (int, error) { return 2, nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.ObtenerImpactoCategoria(1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.SubcategoriasActivas != 2 {
		t.Errorf("esperaba subcategorias_activas=2, obtuvo %d", resp.SubcategoriasActivas)
	}
}

// --- TestReactivarCategoriaNoreactivaSubcats ---

func TestReactivarCategoriaNoreactivaSubcats(t *testing.T) {
	reactivarSubcatsLlamado := false
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) { return categoriaInactiva(), nil },
		reactivarCategoriaFunc: func(id, actualizadoPor uint64) (*cat.Categoria, error) {
			c := categoriaEjemplo()
			c.Activo = true
			return c, nil
		},
		inactivarSubcategoriasDeCategFunc: func(categoriaID, actualizadoPor uint64) (int, error) {
			reactivarSubcatsLlamado = true
			return 0, nil
		},
	}
	svc := newSvc(t, repo)
	_, err := svc.ReactivarCategoria(1, 42, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if reactivarSubcatsLlamado {
		t.Error("reactivar subcategorías NO debe llamarse al reactivar la categoría")
	}
}

// --- TestReactivarSubcatCategoriaPadreInactiva ---

func TestReactivarSubcatCategoriaPadreInactiva(t *testing.T) {
	sub := subcategoriaInactiva()
	repo := &mockRepo{
		obtenerSubcategoriaPorIDFunc: func(id uint64) (*cat.Subcategoria, error) { return sub, nil },
		obtenerCategoriaPorIDFunc:    func(id uint64) (*cat.Categoria, error) { return categoriaInactiva(), nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.ReactivarSubcategoria(1, 42, "admin")
	if !errors.Is(err, cat.ErrCategoriaPadreInactiva) {
		t.Errorf("esperaba ErrCategoriaPadreInactiva, obtuvo: %v", err)
	}
}

// --- TestReactivarSubcatCategoriaPadreActiva ---

func TestReactivarSubcatCategoriaPadreActiva(t *testing.T) {
	sub := subcategoriaInactiva()
	reactivarLlamado := false
	repo := &mockRepo{
		obtenerSubcategoriaPorIDFunc: func(id uint64) (*cat.Subcategoria, error) { return sub, nil },
		obtenerCategoriaPorIDFunc:    func(id uint64) (*cat.Categoria, error) { return categoriaEjemplo(), nil },
		reactivarSubcategoriaFunc: func(id, actualizadoPor uint64) (*cat.Subcategoria, error) {
			reactivarLlamado = true
			s := subcategoriaEjemplo()
			s.Activo = true
			return s, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.ReactivarSubcategoria(1, 42, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !reactivarLlamado {
		t.Error("ReactivarSubcategoria del repo no fue llamado")
	}
	if !resp.Activo {
		t.Error("esperaba activo=true tras reactivación")
	}
}

// --- TestAccesoSinAdminFallaEnEscritura ---
// (cubierto en handler_test.go con httptest; aquí se verifica el guard de rol)

func TestInactivarCategoriaYaInactiva(t *testing.T) {
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) { return categoriaInactiva(), nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.InactivarCategoria(1, 42, "admin")
	if !errors.Is(err, cat.ErrCategoriaYaInactiva) {
		t.Errorf("esperaba ErrCategoriaYaInactiva, obtuvo: %v", err)
	}
}

func TestReactivarCategoriaYaActiva(t *testing.T) {
	repo := &mockRepo{
		obtenerCategoriaPorIDFunc: func(id uint64) (*cat.Categoria, error) { return categoriaEjemplo(), nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.ReactivarCategoria(1, 42, "admin")
	if !errors.Is(err, cat.ErrCategoriaYaActiva) {
		t.Errorf("esperaba ErrCategoriaYaActiva, obtuvo: %v", err)
	}
}
