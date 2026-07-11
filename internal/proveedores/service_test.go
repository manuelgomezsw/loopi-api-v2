package proveedores_test

import (
	"errors"
	"testing"

	pv "github.com/manuelgomezsw/loopi-api-v2/internal/proveedores"
)

func newSvc(t *testing.T, repo pv.Repository) pv.Service {
	t.Helper()
	return pv.NewService(repo)
}

// --- Crear ---

func TestCrearProveedorExitoso(t *testing.T) {
	esperado := proveedorEjemplo()
	repo := &mockRepo{
		existeNITFunc: func(string, *uint64) (bool, error) { return false, nil },
		crearFunc:     func(*pv.CrearProveedorRequest) (*pv.Proveedor, error) { return esperado, nil },
	}
	svc := newSvc(t, repo)
	resultado, err := svc.Crear(&pv.CrearProveedorRequest{
		RazonSocial: "Distribuidora La Cosecha S.A.S", NIT: "900123456-7",
	}, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resultado.ID != esperado.ID {
		t.Errorf("id esperado %d, obtuvo %d", esperado.ID, resultado.ID)
	}
}

func TestCrearProveedorNITDuplicado(t *testing.T) {
	repo := &mockRepo{
		existeNITFunc: func(string, *uint64) (bool, error) { return true, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&pv.CrearProveedorRequest{
		RazonSocial: "Otro Proveedor", NIT: "900123456-7",
	}, 1, "admin")
	if !errors.Is(err, pv.ErrNITDuplicado) {
		t.Errorf("esperaba ErrNITDuplicado, obtuvo: %v", err)
	}
}

func TestCrearProveedorSinRazonSocial(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&pv.CrearProveedorRequest{NIT: "PROV-002"}, 1, "admin")
	var valErr *pv.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "campo_requerido" || valErr.Campo != "razon_social" {
		t.Errorf("esperaba ValidationError campo_requerido/razon_social, obtuvo: %v", err)
	}
}

func TestCrearProveedorSinNIT(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&pv.CrearProveedorRequest{RazonSocial: "Proveedor Simple"}, 1, "admin")
	var valErr *pv.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "campo_requerido" || valErr.Campo != "nit" {
		t.Errorf("esperaba ValidationError campo_requerido/nit, obtuvo: %v", err)
	}
}

func TestCrearProveedorEmailInvalido(t *testing.T) {
	repo := &mockRepo{}
	svc := newSvc(t, repo)
	_, err := svc.Crear(&pv.CrearProveedorRequest{
		RazonSocial: "Proveedor X", NIT: "PROV-003", EmailContacto: strPtr("no-es-un-email"),
	}, 1, "admin")
	var valErr *pv.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "email_invalido" {
		t.Errorf("esperaba ValidationError email_invalido, obtuvo: %v", err)
	}
}

// --- Editar ---

func TestEditarProveedorNITPropioNoConflicto(t *testing.T) {
	p := proveedorEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { return p, nil },
		existeNITFunc:    func(string, *uint64) (bool, error) { return false, nil },
		actualizarFunc: func(id uint64, req *pv.EditarProveedorRequest) (*pv.Proveedor, error) {
			p.NIT = *req.NIT
			return p, nil
		},
	}
	svc := newSvc(t, repo)
	_, err := svc.Editar(p.ID, &pv.EditarProveedorRequest{NIT: strPtr("900123456-7")}, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
}

func TestEditarProveedorNITDuplicado(t *testing.T) {
	p := proveedorEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { return p, nil },
		existeNITFunc:    func(string, *uint64) (bool, error) { return true, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Editar(p.ID, &pv.EditarProveedorRequest{NIT: strPtr("PROV-001")}, 1, "admin")
	if !errors.Is(err, pv.ErrNITDuplicado) {
		t.Errorf("esperaba ErrNITDuplicado, obtuvo: %v", err)
	}
}

func TestEditarProveedorNoExiste(t *testing.T) {
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { return nil, pv.ErrProveedorNoEncontrado },
	}
	svc := newSvc(t, repo)
	_, err := svc.Editar(999, &pv.EditarProveedorRequest{NombreContacto: strPtr("X")}, 1, "admin")
	if !errors.Is(err, pv.ErrProveedorNoEncontrado) {
		t.Errorf("esperaba ErrProveedorNoEncontrado, obtuvo: %v", err)
	}
}

func TestEditarProveedorCampoVacio(t *testing.T) {
	p := proveedorEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { return p, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Editar(p.ID, &pv.EditarProveedorRequest{RazonSocial: strPtr("")}, 1, "admin")
	var valErr *pv.ValidationError
	if !errors.As(err, &valErr) || valErr.Codigo != "campo_vacio" {
		t.Errorf("esperaba ValidationError campo_vacio, obtuvo: %v", err)
	}
}

// --- Inactivar / Activar ---

func TestInactivarProveedor(t *testing.T) {
	p := proveedorEjemplo()
	llamado := false
	repo := &mockRepo{
		obtenerPorIDFunc:  func(uint64) (*pv.Proveedor, error) { return p, nil },
		cambiarEstadoFunc: func(uint64, bool) error { llamado = true; return nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.Inactivar(p.ID, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Activo {
		t.Error("activo debe ser false")
	}
	if !llamado {
		t.Error("repository.CambiarEstado no fue llamado")
	}
}

func TestInactivarYaInactivo(t *testing.T) {
	p := proveedorEjemplo()
	p.Activo = false
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { return p, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Inactivar(p.ID, 1, "admin")
	if !errors.Is(err, pv.ErrYaInactivo) {
		t.Errorf("esperaba ErrYaInactivo, obtuvo: %v", err)
	}
}

func TestActivarProveedor(t *testing.T) {
	p := proveedorEjemplo()
	p.Activo = false
	llamado := false
	repo := &mockRepo{
		obtenerPorIDFunc:  func(uint64) (*pv.Proveedor, error) { return p, nil },
		cambiarEstadoFunc: func(uint64, bool) error { llamado = true; return nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.Activar(p.ID, 1, "admin")
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !resp.Activo {
		t.Error("activo debe ser true")
	}
	if !llamado {
		t.Error("repository.CambiarEstado no fue llamado")
	}
}

func TestActivarYaActivo(t *testing.T) {
	p := proveedorEjemplo()
	repo := &mockRepo{
		obtenerPorIDFunc: func(uint64) (*pv.Proveedor, error) { return p, nil },
	}
	svc := newSvc(t, repo)
	_, err := svc.Activar(p.ID, 1, "admin")
	if !errors.Is(err, pv.ErrYaActivo) {
		t.Errorf("esperaba ErrYaActivo, obtuvo: %v", err)
	}
}

// --- Listar ---

func TestListarFiltroActivo(t *testing.T) {
	repo := &mockRepo{
		listarFunc: func(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
			if filtros.Activo == nil || !*filtros.Activo {
				t.Error("esperaba filtro activo=true propagado")
			}
			return &pv.ListarProveedoresResponse{Proveedores: []pv.Proveedor{*proveedorEjemplo()}, Total: 1, Page: 1, Limit: 50}, nil
		},
	}
	svc := newSvc(t, repo)
	activo := true
	resp, err := svc.Listar(&pv.FiltrosListado{Activo: &activo, Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total esperado 1, obtuvo %d", resp.Total)
	}
}

func TestListarBusquedaPorRazonSocial(t *testing.T) {
	repo := &mockRepo{
		listarFunc: func(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
			if filtros.Busqueda != "cosecha" {
				t.Errorf("esperaba busqueda='cosecha', obtuvo %q", filtros.Busqueda)
			}
			return &pv.ListarProveedoresResponse{Proveedores: []pv.Proveedor{*proveedorEjemplo()}, Total: 1, Page: 1, Limit: 50}, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.Listar(&pv.FiltrosListado{Busqueda: "cosecha", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total esperado 1, obtuvo %d", resp.Total)
	}
}

func TestListarBusquedaPorNIT(t *testing.T) {
	repo := &mockRepo{
		listarFunc: func(filtros *pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
			if filtros.Busqueda != "900123" {
				t.Errorf("esperaba busqueda='900123', obtuvo %q", filtros.Busqueda)
			}
			return &pv.ListarProveedoresResponse{Proveedores: []pv.Proveedor{*proveedorEjemplo()}, Total: 1, Page: 1, Limit: 50}, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.Listar(&pv.FiltrosListado{Busqueda: "900123", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total esperado 1, obtuvo %d", resp.Total)
	}
}

func TestListarEmptyState(t *testing.T) {
	repo := &mockRepo{
		listarFunc: func(*pv.FiltrosListado) (*pv.ListarProveedoresResponse, error) {
			return &pv.ListarProveedoresResponse{Proveedores: []pv.Proveedor{}, Total: 0, Page: 1, Limit: 50}, nil
		},
	}
	svc := newSvc(t, repo)
	resp, err := svc.Listar(&pv.FiltrosListado{Busqueda: "noexiste", Page: 1, Limit: 50})
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 0 || len(resp.Proveedores) != 0 {
		t.Errorf("esperaba total=0 y array vacío, obtuvo total=%d len=%d", resp.Total, len(resp.Proveedores))
	}
}

// --- ObtenerPorID ---

func TestObtenerPorIDConItemsAsignados(t *testing.T) {
	p := proveedorEjemplo()
	detalle := &pv.ProveedorDetalleResponse{Proveedor: *p, ItemsAsignados: 3}
	repo := &mockRepo{
		obtenerPorIDConItemsFunc: func(uint64) (*pv.ProveedorDetalleResponse, error) { return detalle, nil },
	}
	svc := newSvc(t, repo)
	resp, err := svc.ObtenerPorID(p.ID)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.ItemsAsignados != 3 {
		t.Errorf("items_asignados esperado 3, obtuvo %d", resp.ItemsAsignados)
	}
}
