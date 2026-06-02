package empleados

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Errores sentinel del dominio empleados.
var (
	ErrEmpleadoNoEncontrado = errors.New("empleado_no_encontrado")
	ErrUsuarioDuplicado     = errors.New("usuario_duplicado")
	ErrUltimoAdminActivo    = errors.New("ultimo_admin_activo")
	ErrTiendaNoExiste       = errors.New("tienda_no_existe")
	ErrTiendaInactiva       = errors.New("tienda_inactiva")
)

// EmpleadoRepository define todas las operaciones de persistencia para empleados.
type EmpleadoRepository interface {
	ObtenerPorID(ctx context.Context, id uint64) (*Empleado, error)
	ExistePorUsuario(ctx context.Context, usuario string) (bool, error)
	InsertarEmpleado(ctx context.Context, tx *sql.Tx, emp Empleado, hash string) (uint64, error)
	ActualizarEmpleado(ctx context.Context, tx *sql.Tx, id uint64, emp Empleado) error
	ActualizarActivo(ctx context.Context, tx *sql.Tx, id uint64, activo bool) error
	ContarAdminsActivosExcluyendo(ctx context.Context, tx *sql.Tx, id uint64) (int, error)
	ActualizarContrasena(ctx context.Context, id uint64, hash string) error
	MarcarCambioCompletado(ctx context.Context, id uint64) error
	ListarEmpleados(ctx context.Context, p ListarEmpleadosParams) ([]Empleado, int, error)
	ObtenerTiendaActivaPorID(ctx context.Context, id uint64) error
	RegistrarLog(ctx context.Context, tx *sql.Tx, actorID, empleadoID uint64, accion string, detalle map[string]any) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
}

type mysqlEmpleadoRepository struct {
	db *sql.DB
}

// NewRepository crea un nuevo repositorio de empleados.
func NewRepository(db *sql.DB) EmpleadoRepository {
	return &mysqlEmpleadoRepository{db: db}
}

var bogotaLoc = func() *time.Location {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// columnas de SELECT — nunca incluye contrasena_hash (RF-EMP-05.6)
const selectCols = `id, nombre, apellido, usuario, rol, tienda_id,
  tipo_documento, numero_documento, telefono, email, fecha_nacimiento,
  activo, requiere_cambio_contrasena, creado_en, actualizado_en`

func (r *mysqlEmpleadoRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// ObtenerPorID recupera un empleado por PK (sin contrasena_hash).
func (r *mysqlEmpleadoRepository) ObtenerPorID(ctx context.Context, id uint64) (*Empleado, error) {
	return r.scanEmpleado(r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM empleados WHERE id = ?", selectCols), id,
	))
}

// ExistePorUsuario retorna true si ya existe un empleado con ese nombre de usuario.
func (r *mysqlEmpleadoRepository) ExistePorUsuario(ctx context.Context, usuario string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(1) FROM empleados WHERE usuario = ?", usuario,
	).Scan(&count)
	return count > 0, err
}

// InsertarEmpleado crea un nuevo empleado dentro de la transacción dada.
func (r *mysqlEmpleadoRepository) InsertarEmpleado(ctx context.Context, tx *sql.Tx, emp Empleado, hash string) (uint64, error) {
	ahora := time.Now().In(bogotaLoc)
	result, err := tx.ExecContext(ctx,
		`INSERT INTO empleados
		  (nombre, apellido, usuario, contrasena_hash, rol, tienda_id,
		   tipo_documento, numero_documento, telefono, email, fecha_nacimiento,
		   activo, requiere_cambio_contrasena, creado_en, actualizado_en)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1, ?, ?)`,
		emp.Nombre, emp.Apellido, emp.Usuario, hash, emp.Rol,
		toNullUint64(emp.TiendaID),
		toNullString(emp.TipoDocumento), toNullString(emp.NumeroDocumento),
		toNullString(emp.Telefono), toNullString(emp.Email),
		toNullTime(emp.FechaNacimiento),
		ahora, ahora,
	)
	if err != nil {
		return 0, mapMySQLError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("obtener id de empleado creado: %w", err)
	}
	return uint64(id), nil
}

// ActualizarEmpleado actualiza los campos mutables del empleado dentro de una transacción.
// El campo usuario nunca se actualiza (RF-EMP-02.2).
func (r *mysqlEmpleadoRepository) ActualizarEmpleado(ctx context.Context, tx *sql.Tx, id uint64, emp Empleado) error {
	ahora := time.Now().In(bogotaLoc)
	_, err := tx.ExecContext(ctx,
		`UPDATE empleados
		 SET nombre = ?, apellido = ?, rol = ?, tienda_id = ?,
		     tipo_documento = ?, numero_documento = ?, telefono = ?,
		     email = ?, fecha_nacimiento = ?, actualizado_en = ?
		 WHERE id = ?`,
		emp.Nombre, emp.Apellido, emp.Rol,
		toNullUint64(emp.TiendaID),
		toNullString(emp.TipoDocumento), toNullString(emp.NumeroDocumento),
		toNullString(emp.Telefono), toNullString(emp.Email),
		toNullTime(emp.FechaNacimiento),
		ahora, id,
	)
	return err
}

// ActualizarActivo cambia el estado activo/inactivo del empleado dentro de una transacción.
func (r *mysqlEmpleadoRepository) ActualizarActivo(ctx context.Context, tx *sql.Tx, id uint64, activo bool) error {
	activoInt := 0
	if activo {
		activoInt = 1
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE empleados SET activo = ?, actualizado_en = ? WHERE id = ?",
		activoInt, time.Now().In(bogotaLoc), id,
	)
	return err
}

// ContarAdminsActivosExcluyendo retorna cuántos admins activos hay excluyendo el id dado.
// Debe ejecutarse dentro de una transacción para garantizar atomicidad (RD-04).
func (r *mysqlEmpleadoRepository) ContarAdminsActivosExcluyendo(ctx context.Context, tx *sql.Tx, id uint64) (int, error) {
	var count int
	err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM empleados WHERE rol = 'admin' AND activo = 1 AND id != ?", id,
	).Scan(&count)
	return count, err
}

// ActualizarContrasena actualiza el hash y activa el flag requiere_cambio_contrasena.
func (r *mysqlEmpleadoRepository) ActualizarContrasena(ctx context.Context, id uint64, hash string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE empleados SET contrasena_hash = ?, requiere_cambio_contrasena = 1, actualizado_en = ? WHERE id = ?",
		hash, time.Now().In(bogotaLoc), id,
	)
	return err
}

// MarcarCambioCompletado pone requiere_cambio_contrasena = 0 tras cambio exitoso.
func (r *mysqlEmpleadoRepository) MarcarCambioCompletado(ctx context.Context, id uint64) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE empleados SET requiere_cambio_contrasena = 0, actualizado_en = ? WHERE id = ?",
		time.Now().In(bogotaLoc), id,
	)
	return err
}

// ListarEmpleados retorna empleados paginados con búsqueda y filtros.
func (r *mysqlEmpleadoRepository) ListarEmpleados(ctx context.Context, p ListarEmpleadosParams) ([]Empleado, int, error) {
	offset := (p.Page - 1) * p.Limit

	var where []string
	var args []any

	if p.Q != "" {
		where = append(where, `(LOWER(CONCAT(nombre, ' ', apellido)) LIKE LOWER(CONCAT('%', ?, '%'))
		  OR LOWER(usuario) LIKE LOWER(CONCAT('%', ?, '%')))`)
		args = append(args, p.Q, p.Q)
	}
	if p.TiendaID != nil {
		where = append(where, "tienda_id = ?")
		args = append(args, *p.TiendaID)
	}
	if p.Activo != nil {
		activoInt := 0
		if *p.Activo {
			activoInt = 1
		}
		where = append(where, "activo = ?")
		args = append(args, activoInt)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(1) FROM empleados %s", whereClause),
		countArgs...,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	listArgs := append(args, p.Limit, offset)
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf(
			"SELECT %s FROM empleados %s ORDER BY apellido ASC, nombre ASC LIMIT ? OFFSET ?",
			selectCols, whereClause,
		),
		listArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var empleados []Empleado
	for rows.Next() {
		e, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		empleados = append(empleados, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if empleados == nil {
		empleados = []Empleado{}
	}
	return empleados, total, nil
}

// ObtenerTiendaActivaPorID verifica que la tienda exista y esté activa.
// Retorna ErrTiendaNoExiste o ErrTiendaInactiva según el caso.
func (r *mysqlEmpleadoRepository) ObtenerTiendaActivaPorID(ctx context.Context, id uint64) error {
	var activo int
	err := r.db.QueryRowContext(ctx,
		"SELECT activo FROM tiendas WHERE id = ?", id,
	).Scan(&activo)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTiendaNoExiste
	}
	if err != nil {
		return err
	}
	if activo == 0 {
		return ErrTiendaInactiva
	}
	return nil
}

// RegistrarLog inserta un registro en log_auditoria_empleados.
// El campo detalle nunca debe incluir contraseñas ni hashes (RF-EMP-05-A.2).
func (r *mysqlEmpleadoRepository) RegistrarLog(ctx context.Context, tx *sql.Tx, actorID, empleadoID uint64, accion string, detalle map[string]any) error {
	detalleJSON, err := json.Marshal(detalle)
	if err != nil {
		return fmt.Errorf("serializar detalle de audit log: %w", err)
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO log_auditoria_empleados (actor_id, accion, empleado_id, detalle, creado_en)
		 VALUES (?, ?, ?, ?, ?)`,
		actorID, accion, empleadoID, detalleJSON, time.Now().In(bogotaLoc),
	)
	return err
}

// --- helpers privados ---

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *mysqlEmpleadoRepository) scanEmpleado(row rowScanner) (*Empleado, error) {
	var e Empleado
	var activo, requiere int
	var tiendaID sql.NullInt64
	var tipoDoc, numDoc, tel, email sql.NullString
	var fechaNac sql.NullTime

	err := row.Scan(
		&e.ID, &e.Nombre, &e.Apellido, &e.Usuario, &e.Rol, &tiendaID,
		&tipoDoc, &numDoc, &tel, &email, &fechaNac,
		&activo, &requiere, &e.CreadoEn, &e.ActualizadoEn,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEmpleadoNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	e.Activo = activo == 1
	e.RequiereCambioContrasena = requiere == 1
	if tiendaID.Valid {
		v := uint64(tiendaID.Int64)
		e.TiendaID = &v
	}
	if tipoDoc.Valid {
		e.TipoDocumento = &tipoDoc.String
	}
	if numDoc.Valid {
		e.NumeroDocumento = &numDoc.String
	}
	if tel.Valid {
		e.Telefono = &tel.String
	}
	if email.Valid {
		e.Email = &email.String
	}
	if fechaNac.Valid {
		e.FechaNacimiento = &fechaNac.Time
	}
	return &e, nil
}

func (r *mysqlEmpleadoRepository) scanRow(rows *sql.Rows) (Empleado, error) {
	var e Empleado
	var activo, requiere int
	var tiendaID sql.NullInt64
	var tipoDoc, numDoc, tel, email sql.NullString
	var fechaNac sql.NullTime

	err := rows.Scan(
		&e.ID, &e.Nombre, &e.Apellido, &e.Usuario, &e.Rol, &tiendaID,
		&tipoDoc, &numDoc, &tel, &email, &fechaNac,
		&activo, &requiere, &e.CreadoEn, &e.ActualizadoEn,
	)
	if err != nil {
		return Empleado{}, err
	}
	e.Activo = activo == 1
	e.RequiereCambioContrasena = requiere == 1
	if tiendaID.Valid {
		v := uint64(tiendaID.Int64)
		e.TiendaID = &v
	}
	if tipoDoc.Valid {
		e.TipoDocumento = &tipoDoc.String
	}
	if numDoc.Valid {
		e.NumeroDocumento = &numDoc.String
	}
	if tel.Valid {
		e.Telefono = &tel.String
	}
	if email.Valid {
		e.Email = &email.String
	}
	if fechaNac.Valid {
		e.FechaNacimiento = &fechaNac.Time
	}
	return e, nil
}

func toNullUint64(v *uint64) any {
	if v == nil {
		return nil
	}
	return *v
}

func toNullString(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func toNullTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.Format("2006-01-02")
}

func mapMySQLError(err error) error {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		if strings.Contains(mysqlErr.Message, "uq_empleados_usuario") {
			return ErrUsuarioDuplicado
		}
	}
	return err
}
