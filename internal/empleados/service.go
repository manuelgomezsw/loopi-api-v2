package empleados

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/manuelgomezsw/loopi-api-v2/config"
)

// EmpleadoService define las operaciones de negocio para empleados.
type EmpleadoService interface {
	CrearEmpleado(ctx context.Context, actorID uint64, req CrearEmpleadoRequest) (*CrearEmpleadoResponse, error)
	ObtenerEmpleado(ctx context.Context, id uint64) (*EmpleadoResponse, error)
	EditarEmpleado(ctx context.Context, actorID, empleadoID uint64, req EditarEmpleadoRequest) (*EmpleadoResponse, error)
	CambiarEstado(ctx context.Context, actorID, empleadoID uint64, activo bool) (*EmpleadoResponse, error)
	ListarEmpleados(ctx context.Context, p ListarEmpleadosParams) (*ListarEmpleadosResponse, error)
	ResetearContrasena(ctx context.Context, actorID, empleadoID uint64) (string, error)
	CambiarContrasena(ctx context.Context, empleadoID uint64, nuevaContrasena string) error
}

type empleadoService struct {
	repo        EmpleadoRepository
	bcryptCost  int
}

// NewService crea un nuevo EmpleadoService.
func NewService(repo EmpleadoRepository) EmpleadoService {
	return &empleadoService{repo: repo, bcryptCost: config.BcryptCostProd}
}

// NewServiceWithCost crea el servicio con un factor bcrypt configurable (usado en tests con cost 4).
func NewServiceWithCost(repo EmpleadoRepository, cost int) EmpleadoService {
	return &empleadoService{repo: repo, bcryptCost: cost}
}

// rolesConTienda son los roles que requieren tienda_id.
var rolesConTienda = map[string]bool{"lider_tienda": true, "barista": true}

// CrearEmpleado registra un nuevo empleado con contraseña temporal.
func (s *empleadoService) CrearEmpleado(ctx context.Context, actorID uint64, req CrearEmpleadoRequest) (*CrearEmpleadoResponse, error) {
	// Validar campos obligatorios.
	if req.Nombre == "" || req.Apellido == "" || req.Usuario == "" || req.Rol == "" {
		return nil, &ValidationError{Codigo: "campo_requerido", Mensaje: "Nombre, apellido, usuario y rol son obligatorios."}
	}

	// Validar rol.
	if req.Rol != "admin" && !rolesConTienda[req.Rol] {
		return nil, &ValidationError{Codigo: "rol_invalido", Mensaje: "El rol debe ser admin, lider_tienda o barista.", Campo: "rol"}
	}

	// Validar tienda según rol.
	if rolesConTienda[req.Rol] {
		if req.TiendaID == nil {
			return nil, &ValidationError{Codigo: "tienda_requerida", Mensaje: "La tienda es obligatoria para baristas y líderes de tienda.", Campo: "tienda_id"}
		}
		if err := s.repo.ObtenerTiendaActivaPorID(ctx, *req.TiendaID); err != nil {
			if errors.Is(err, ErrTiendaNoExiste) || errors.Is(err, ErrTiendaInactiva) {
				return nil, &ValidationError{Codigo: "tienda_no_existe", Mensaje: "La tienda no existe o está inactiva.", Campo: "tienda_id"}
			}
			return nil, err
		}
	}
	if req.Rol == "admin" && req.TiendaID != nil {
		return nil, &ValidationError{Codigo: "tienda_no_permitida_para_admin", Mensaje: "El rol admin no puede tener tienda asignada.", Campo: "tienda_id"}
	}

	// Verificar unicidad del usuario.
	existe, err := s.repo.ExistePorUsuario(ctx, req.Usuario)
	if err != nil {
		return nil, err
	}
	if existe {
		return nil, ErrUsuarioDuplicado
	}

	// Generar y hashear contraseña temporal.
	contrasenaTemp, err := generarContrasenaTemp()
	if err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(contrasenaTemp), s.bcryptCost)
	if err != nil {
		return nil, err
	}

	emp := Empleado{
		Nombre:          req.Nombre,
		Apellido:        req.Apellido,
		Usuario:         req.Usuario,
		Rol:             req.Rol,
		TiendaID:        req.TiendaID,
		TipoDocumento:   req.TipoDocumento,
		NumeroDocumento: req.NumeroDocumento,
		Telefono:        req.Telefono,
		Email:           req.Email,
	}
	if req.FechaNacimiento != nil {
		// Se almacena como string "YYYY-MM-DD"; el repo lo pasa tal cual.
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	id, err := s.repo.InsertarEmpleado(ctx, tx, emp, string(hash))
	if err != nil {
		return nil, err
	}

	camposNuevos := map[string]any{"rol": req.Rol, "nombre": req.Nombre, "apellido": req.Apellido}
	if req.TiendaID != nil {
		camposNuevos["tienda_id"] = *req.TiendaID
	}
	if err := s.repo.RegistrarLog(ctx, tx, actorID, id, "CREAR", map[string]any{"campos_nuevos": camposNuevos}); err != nil {
		return nil, err
	}

	if err := commitTx(tx); err != nil {
		return nil, err
	}

	creado, err := s.repo.ObtenerPorID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &CrearEmpleadoResponse{
		EmpleadoResponse:   toResponse(*creado),
		ContrasenaTemporal: contrasenaTemp,
	}, nil
}

// ObtenerEmpleado retorna el detalle de un empleado.
func (s *empleadoService) ObtenerEmpleado(ctx context.Context, id uint64) (*EmpleadoResponse, error) {
	e, err := s.repo.ObtenerPorID(ctx, id)
	if err != nil {
		return nil, err
	}
	r := toResponse(*e)
	return &r, nil
}

// EditarEmpleado actualiza los campos enviados en el request.
func (s *empleadoService) EditarEmpleado(ctx context.Context, actorID, empleadoID uint64, req EditarEmpleadoRequest) (*EmpleadoResponse, error) {
	actual, err := s.repo.ObtenerPorID(ctx, empleadoID)
	if err != nil {
		return nil, err
	}

	anterior := map[string]any{"rol": actual.Rol}
	if actual.TiendaID != nil {
		anterior["tienda_id"] = *actual.TiendaID
	}

	// Aplicar cambios del request sobre el empleado actual.
	updated := *actual
	if req.Nombre != nil {
		updated.Nombre = *req.Nombre
	}
	if req.Apellido != nil {
		updated.Apellido = *req.Apellido
	}
	if req.TipoDocumento != nil {
		updated.TipoDocumento = req.TipoDocumento
	}
	if req.NumeroDocumento != nil {
		updated.NumeroDocumento = req.NumeroDocumento
	}
	if req.Telefono != nil {
		updated.Telefono = req.Telefono
	}
	if req.Email != nil {
		updated.Email = req.Email
	}

	rolCambia := req.Rol != nil && *req.Rol != actual.Rol
	if req.Rol != nil {
		updated.Rol = *req.Rol
	}

	// Si el rol cambia a admin, limpiar tienda (RF-EMP-02.4).
	if updated.Rol == "admin" {
		updated.TiendaID = nil
	} else if rolesConTienda[updated.Rol] {
		// Si el nuevo rol requiere tienda, validarla.
		if req.TiendaID != nil {
			updated.TiendaID = req.TiendaID
		}
		if updated.TiendaID == nil {
			return nil, &ValidationError{Codigo: "tienda_requerida", Mensaje: "La tienda es obligatoria para baristas y líderes de tienda.", Campo: "tienda_id"}
		}
		if err := s.repo.ObtenerTiendaActivaPorID(ctx, *updated.TiendaID); err != nil {
			if errors.Is(err, ErrTiendaNoExiste) || errors.Is(err, ErrTiendaInactiva) {
				return nil, &ValidationError{Codigo: "tienda_no_existe", Mensaje: "La tienda no existe o está inactiva.", Campo: "tienda_id"}
			}
			return nil, err
		}
	} else if req.TiendaID != nil {
		updated.TiendaID = req.TiendaID
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	// Si el rol cambia desde admin, verificar que no sea el último admin activo (RF-EMP-02.6).
	if rolCambia && actual.Rol == "admin" && updated.Rol != "admin" {
		count, err := s.repo.ContarAdminsActivosExcluyendo(ctx, tx, empleadoID)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrUltimoAdminActivo
		}
	}

	if err := s.repo.ActualizarEmpleado(ctx, tx, empleadoID, updated); err != nil {
		return nil, err
	}

	nuevo := map[string]any{"rol": updated.Rol}
	if updated.TiendaID != nil {
		nuevo["tienda_id"] = *updated.TiendaID
	}
	detalle := map[string]any{"campos_anteriores": anterior, "campos_nuevos": nuevo}
	if err := s.repo.RegistrarLog(ctx, tx, actorID, empleadoID, "EDITAR", detalle); err != nil {
		return nil, err
	}

	if err := commitTx(tx); err != nil {
		return nil, err
	}

	e, err := s.repo.ObtenerPorID(ctx, empleadoID)
	if err != nil {
		return nil, err
	}
	r := toResponse(*e)
	return &r, nil
}

// CambiarEstado activa o inactiva un empleado.
func (s *empleadoService) CambiarEstado(ctx context.Context, actorID, empleadoID uint64, activo bool) (*EmpleadoResponse, error) {
	actual, err := s.repo.ObtenerPorID(ctx, empleadoID)
	if err != nil {
		return nil, err
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackTx(tx)

	// Proteger al último admin activo (RF-EMP-03.5).
	if !activo && actual.Rol == "admin" {
		count, err := s.repo.ContarAdminsActivosExcluyendo(ctx, tx, empleadoID)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrUltimoAdminActivo
		}
	}

	if err := s.repo.ActualizarActivo(ctx, tx, empleadoID, activo); err != nil {
		return nil, err
	}

	accion := "REACTIVAR"
	if !activo {
		accion = "INACTIVAR"
	}
	detalle := map[string]any{"estado_anterior": actual.Activo, "estado_nuevo": activo}
	if err := s.repo.RegistrarLog(ctx, tx, actorID, empleadoID, accion, detalle); err != nil {
		return nil, err
	}

	if err := commitTx(tx); err != nil {
		return nil, err
	}

	e, err := s.repo.ObtenerPorID(ctx, empleadoID)
	if err != nil {
		return nil, err
	}
	r := toResponse(*e)
	return &r, nil
}

// ListarEmpleados retorna la lista paginada con validación de parámetros.
func (s *empleadoService) ListarEmpleados(ctx context.Context, p ListarEmpleadosParams) (*ListarEmpleadosResponse, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > 100 {
		p.Limit = 20
	}
	empleados, total, err := s.repo.ListarEmpleados(ctx, p)
	if err != nil {
		return nil, err
	}
	items := make([]EmpleadoResponse, len(empleados))
	for i, e := range empleados {
		items[i] = toResponse(e)
	}
	return &ListarEmpleadosResponse{
		Empleados: items,
		Total:     total,
		Page:      p.Page,
		Limit:     p.Limit,
	}, nil
}

// ResetearContrasena genera una nueva contraseña temporal y la persiste hasheada.
func (s *empleadoService) ResetearContrasena(ctx context.Context, actorID, empleadoID uint64) (string, error) {
	if _, err := s.repo.ObtenerPorID(ctx, empleadoID); err != nil {
		return "", err
	}

	contrasenaTemp, err := generarContrasenaTemp()
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(contrasenaTemp), s.bcryptCost)
	if err != nil {
		return "", err
	}

	if err := s.repo.ActualizarContrasena(ctx, empleadoID, string(hash)); err != nil {
		return "", err
	}

	// Audit log fuera de TX — RESET_CONTRASENA nunca incluye la contraseña (RF-EMP-05-A.2).
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return "", err
	}
	defer rollbackTx(tx)
	if err := s.repo.RegistrarLog(ctx, tx, actorID, empleadoID, "RESET_CONTRASENA", map[string]any{"motivo": "reset_admin"}); err != nil {
		return "", err
	}
	if err := commitTx(tx); err != nil {
		return "", err
	}

	return contrasenaTemp, nil
}

// CambiarContrasena valida y persiste la nueva contraseña del empleado (RF-EMP-04.5).
func (s *empleadoService) CambiarContrasena(ctx context.Context, empleadoID uint64, nuevaContrasena string) error {
	if len(nuevaContrasena) < 4 {
		return &ValidationError{Codigo: "contrasena_muy_corta", Mensaje: "La contraseña debe tener al menos 4 caracteres.", Campo: "nueva_contrasena"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(nuevaContrasena), s.bcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.ActualizarContrasena(ctx, empleadoID, string(hash)); err != nil {
		return err
	}
	return s.repo.MarcarCambioCompletado(ctx, empleadoID)
}

// commitTx hace commit de una tx no nula.
func commitTx(tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	return tx.Commit()
}

// rollbackTx hace rollback de una tx no nula (seguro llamar tras commit).
func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

// generarContrasenaTemp genera 9 bytes aleatorios → 12 chars base64 URL-safe (RD-01).
func generarContrasenaTemp() (string, error) {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidationError es un error de validación con campo opcional.
type ValidationError struct {
	Codigo  string
	Mensaje string
	Campo   string
}

func (e *ValidationError) Error() string { return e.Codigo }
