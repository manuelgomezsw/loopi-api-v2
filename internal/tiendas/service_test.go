package tiendas_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/manuelgomezsw/loopi-api-v2/internal/tiendas"
)

// --- Listar ---

func TestService_Listar_EstadoInvalido_RetornaValidationError(t *testing.T) {
	repo := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			return nil, 0, nil
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Listar("xnvalido", 1, 50)
	if err == nil {
		t.Fatal("esperaba error por estado inválido")
	}
	var valErr *tiendas.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("esperaba ValidationError, obtuvo: %T", err)
	}
	if valErr.Campo != "estado" {
		t.Errorf("campo esperado 'estado', obtuvo %q", valErr.Campo)
	}
}

func TestService_Listar_PaginaYLimiteDefaults_Normalizados(t *testing.T) {
	var paginaCapturada, limiteCapturado int
	repo := &mockRepo{
		listarFunc: func(estado string, pagina, limite int) ([]tiendas.Tienda, int, error) {
			paginaCapturada = pagina
			limiteCapturado = limite
			return []tiendas.Tienda{}, 0, nil
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Listar("todos", 0, 200)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if paginaCapturada != 1 {
		t.Errorf("pagina debe normalizarse a 1, obtuvo %d", paginaCapturada)
	}
	if limiteCapturado != 50 {
		t.Errorf("limite >100 debe normalizarse a 50, obtuvo %d", limiteCapturado)
	}
}

func TestService_Listar_OK_RetornaListaResponse(t *testing.T) {
	ejemplo := tiendaEjemplo()
	repo := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			return []tiendas.Tienda{ejemplo}, 1, nil
		},
	}
	svc := tiendas.NewService(repo)

	resp, err := svc.Listar("activo", 1, 50)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total esperado 1, obtuvo %d", resp.Total)
	}
	if len(resp.Datos) != 1 {
		t.Fatalf("esperaba 1 dato, obtuvo %d", len(resp.Datos))
	}
	if resp.Datos[0].Codigo != "TDA-001" {
		t.Errorf("codigo inesperado: %q", resp.Datos[0].Codigo)
	}
}

func TestService_Listar_ErrorRepo_Propaga(t *testing.T) {
	errBD := errors.New("fallo de BD")
	repo := &mockRepo{
		listarFunc: func(string, int, int) ([]tiendas.Tienda, int, error) {
			return nil, 0, errBD
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Listar("todos", 1, 50)
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- Crear ---

func TestService_Crear_NormalizaCodigoMayusculas(t *testing.T) {
	var codigoGuardado string
	repo := &mockRepo{
		crearFunc: func(t tiendas.Tienda) (tiendas.Tienda, error) {
			codigoGuardado = t.Codigo
			return t, nil
		},
	}
	svc := tiendas.NewService(repo)

	req := tiendas.TiendaRequest{
		Codigo: "tda-001", Nombre: "Norte", Direccion: "Calle 1",
		Ciudad: "Bogotá", Telefono: "300",
	}
	_, err := svc.Crear(req, 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if codigoGuardado != "TDA-001" {
		t.Errorf("codigo debe guardarse en mayúsculas, obtuvo %q", codigoGuardado)
	}
}

func TestService_Crear_AsignaAdminIDYTimestamps(t *testing.T) {
	var creadoPor, actualizadoPor uint64
	var creadoEn time.Time
	repo := &mockRepo{
		crearFunc: func(t tiendas.Tienda) (tiendas.Tienda, error) {
			creadoPor = t.CreadoPor
			actualizadoPor = t.ActualizadoPor
			creadoEn = t.CreadoEn
			return t, nil
		},
	}
	svc := tiendas.NewService(repo)

	antes := time.Now()
	_, err := svc.Crear(tiendas.TiendaRequest{
		Codigo: "X", Nombre: "Y", Direccion: "Z", Ciudad: "W", Telefono: "1",
	}, 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if creadoPor != 42 {
		t.Errorf("creado_por esperado 42, obtuvo %d", creadoPor)
	}
	if actualizadoPor != 42 {
		t.Errorf("actualizado_por esperado 42, obtuvo %d", actualizadoPor)
	}
	if creadoEn.Before(antes) {
		t.Error("creado_en debe ser >= al inicio del test")
	}
}

func TestService_Crear_NombreDuplicado_PropagaError(t *testing.T) {
	repo := &mockRepo{
		crearFunc: func(t tiendas.Tienda) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrNombreDuplicado
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Crear(tiendas.TiendaRequest{
		Codigo: "X", Nombre: "Existente", Direccion: "D", Ciudad: "C", Telefono: "T",
	}, 1)
	if !errors.Is(err, tiendas.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

func TestService_Crear_CodigoDuplicado_PropagaError(t *testing.T) {
	repo := &mockRepo{
		crearFunc: func(t tiendas.Tienda) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrCodigoDuplicado
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Crear(tiendas.TiendaRequest{
		Codigo: "DUP", Nombre: "N", Direccion: "D", Ciudad: "C", Telefono: "T",
	}, 1)
	if !errors.Is(err, tiendas.ErrCodigoDuplicado) {
		t.Errorf("esperaba ErrCodigoDuplicado, obtuvo: %v", err)
	}
}

// --- ObtenerPorID ---

func TestService_ObtenerPorID_NoExiste_Propaga404(t *testing.T) {
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrTiendaNoEncontrada
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.ObtenerPorID(999)
	if !errors.Is(err, tiendas.ErrTiendaNoEncontrada) {
		t.Errorf("esperaba ErrTiendaNoEncontrada, obtuvo: %v", err)
	}
}

func TestService_ObtenerPorID_OK_RetornaResponse(t *testing.T) {
	ejemplo := tiendaEjemplo()
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return ejemplo, nil
		},
	}
	svc := tiendas.NewService(repo)

	resp, err := svc.ObtenerPorID(1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Codigo != "TDA-001" {
		t.Errorf("codigo inesperado: %q", resp.Codigo)
	}
}

// --- Actualizar ---

func TestService_Actualizar_TiendaNoExiste_Propaga404(t *testing.T) {
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrTiendaNoEncontrada
		},
		actualizarFunc: func(uint64, tiendas.TiendaUpdateRequest, uint64, time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, nil
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Actualizar(999, tiendas.TiendaUpdateRequest{Nombre: "N", Direccion: "D", Ciudad: "C", Telefono: "T"}, 1)
	if !errors.Is(err, tiendas.ErrTiendaNoEncontrada) {
		t.Errorf("esperaba ErrTiendaNoEncontrada, obtuvo: %v", err)
	}
}

func TestService_Actualizar_NombreDuplicado_PropagaError(t *testing.T) {
	ejemplo := tiendaEjemplo()
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return ejemplo, nil
		},
		actualizarFunc: func(uint64, tiendas.TiendaUpdateRequest, uint64, time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrNombreDuplicado
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Actualizar(1, tiendas.TiendaUpdateRequest{Nombre: "Otro", Direccion: "D", Ciudad: "C", Telefono: "T"}, 1)
	if !errors.Is(err, tiendas.ErrNombreDuplicado) {
		t.Errorf("esperaba ErrNombreDuplicado, obtuvo: %v", err)
	}
}

func TestService_Actualizar_OK_CodigoIntacto(t *testing.T) {
	ejemplo := tiendaEjemplo() // Codigo = "TDA-001"
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return ejemplo, nil
		},
		actualizarFunc: func(id uint64, req tiendas.TiendaUpdateRequest, adminID uint64, _ time.Time) (tiendas.Tienda, error) {
			ejemplo.Nombre = req.Nombre
			return ejemplo, nil
		},
	}
	svc := tiendas.NewService(repo)

	resp, err := svc.Actualizar(1, tiendas.TiendaUpdateRequest{Nombre: "Nuevo", Direccion: "D", Ciudad: "C", Telefono: "T"}, 1)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Codigo != "TDA-001" {
		t.Errorf("codigo debe mantenerse intacto, obtuvo %q", resp.Codigo)
	}
	if resp.Nombre != "Nuevo" {
		t.Errorf("nombre no actualizado: %q", resp.Nombre)
	}
}

// --- Inactivar ---

func TestService_Inactivar_TiendaNoExiste_Propaga404(t *testing.T) {
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrTiendaNoEncontrada
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Inactivar(999, 1)
	if !errors.Is(err, tiendas.ErrTiendaNoEncontrada) {
		t.Errorf("esperaba ErrTiendaNoEncontrada, obtuvo: %v", err)
	}
}

func TestService_Inactivar_YaInactiva_RetornaError(t *testing.T) {
	inactiva := tiendaEjemplo()
	inactiva.Activo = false
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return inactiva, nil
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Inactivar(1, 1)
	if !errors.Is(err, tiendas.ErrTiendaYaInactiva) {
		t.Errorf("esperaba ErrTiendaYaInactiva, obtuvo: %v", err)
	}
}

func TestService_Inactivar_OK_ActivoPasaAFalse(t *testing.T) {
	activa := tiendaEjemplo()
	activa.Activo = true
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return activa, nil
		},
		cambiarActivoFunc: func(id uint64, activo bool, adminID uint64, _ time.Time) (tiendas.Tienda, error) {
			activa.Activo = activo
			return activa, nil
		},
	}
	svc := tiendas.NewService(repo)

	resp, err := svc.Inactivar(1, 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if resp.Activo {
		t.Error("activo debe ser false tras inactivar")
	}
}

// --- Reactivar ---

func TestService_Reactivar_TiendaNoExiste_Propaga404(t *testing.T) {
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, tiendas.ErrTiendaNoEncontrada
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Reactivar(999, 1)
	if !errors.Is(err, tiendas.ErrTiendaNoEncontrada) {
		t.Errorf("esperaba ErrTiendaNoEncontrada, obtuvo: %v", err)
	}
}

func TestService_Reactivar_YaActiva_RetornaError(t *testing.T) {
	activa := tiendaEjemplo()
	activa.Activo = true
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return activa, nil
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Reactivar(1, 1)
	if !errors.Is(err, tiendas.ErrTiendaYaActiva) {
		t.Errorf("esperaba ErrTiendaYaActiva, obtuvo: %v", err)
	}
}

func TestService_Reactivar_OK_ActivoPasaATrue(t *testing.T) {
	inactiva := tiendaEjemplo()
	inactiva.Activo = false
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return inactiva, nil
		},
		cambiarActivoFunc: func(id uint64, activo bool, adminID uint64, _ time.Time) (tiendas.Tienda, error) {
			inactiva.Activo = activo
			return inactiva, nil
		},
	}
	svc := tiendas.NewService(repo)

	resp, err := svc.Reactivar(1, 42)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !resp.Activo {
		t.Error("activo debe ser true tras reactivar")
	}
}

func TestService_Inactivar_ErrorCambiarActivo_Propaga(t *testing.T) {
	activa := tiendaEjemplo()
	errBD := errors.New("bd caída")
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return activa, nil
		},
		cambiarActivoFunc: func(uint64, bool, uint64, time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, errBD
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Inactivar(1, 42)
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

func TestService_Reactivar_ErrorCambiarActivo_Propaga(t *testing.T) {
	inactiva := tiendaEjemplo()
	inactiva.Activo = false
	errBD := errors.New("bd caída")
	repo := &mockRepo{
		obtenerFunc: func(id uint64) (tiendas.Tienda, error) {
			return inactiva, nil
		},
		cambiarActivoFunc: func(uint64, bool, uint64, time.Time) (tiendas.Tienda, error) {
			return tiendas.Tienda{}, errBD
		},
	}
	svc := tiendas.NewService(repo)

	_, err := svc.Reactivar(1, 42)
	if !errors.Is(err, errBD) {
		t.Errorf("esperaba errBD, obtuvo: %v", err)
	}
}

// --- ValidationError ---

func TestValidationError_ErrorString_ContieneCodigo(t *testing.T) {
	err := &tiendas.ValidationError{Codigo: "estado_invalido", Mensaje: "m", Campo: "estado"}
	if !strings.Contains(err.Error(), "estado_invalido") {
		t.Errorf("Error() debe contener el código, obtuvo: %q", err.Error())
	}
}
