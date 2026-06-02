package empleados

import "time"

// Empleado es la entidad interna — incluye contrasena_hash para operaciones del servicio.
// NUNCA exponer contrasena_hash en respuestas de API (RF-EMP-05.6).
type Empleado struct {
	ID                       uint64
	Nombre                   string
	Apellido                 string
	Usuario                  string
	ContrasenaHash           string // interno; nunca en responses
	Rol                      string
	TiendaID                 *uint64
	TipoDocumento            *string
	NumeroDocumento          *string
	Telefono                 *string
	Email                    *string
	FechaNacimiento          *time.Time
	Activo                   bool
	RequiereCambioContrasena bool
	CreadoEn                 time.Time
	ActualizadoEn            time.Time
}

// CrearEmpleadoRequest es el body del POST /api/v1/empleados.
type CrearEmpleadoRequest struct {
	Nombre          string  `json:"nombre"`
	Apellido        string  `json:"apellido"`
	Usuario         string  `json:"usuario"`
	Rol             string  `json:"rol"`
	TiendaID        *uint64 `json:"tienda_id,omitempty"`
	TipoDocumento   *string `json:"tipo_documento,omitempty"`
	NumeroDocumento *string `json:"numero_documento,omitempty"`
	Telefono        *string `json:"telefono,omitempty"`
	Email           *string `json:"email,omitempty"`
	FechaNacimiento *string `json:"fecha_nacimiento,omitempty"` // "YYYY-MM-DD"
}

// EditarEmpleadoRequest es el body del PUT /api/v1/empleados/{id}.
// Todos los campos son opcionales; solo se actualizan los enviados.
type EditarEmpleadoRequest struct {
	Nombre          *string `json:"nombre,omitempty"`
	Apellido        *string `json:"apellido,omitempty"`
	Rol             *string `json:"rol,omitempty"`
	TiendaID        *uint64 `json:"tienda_id,omitempty"`
	TipoDocumento   *string `json:"tipo_documento,omitempty"`
	NumeroDocumento *string `json:"numero_documento,omitempty"`
	Telefono        *string `json:"telefono,omitempty"`
	Email           *string `json:"email,omitempty"`
	FechaNacimiento *string `json:"fecha_nacimiento,omitempty"`
}

// CambiarEstadoRequest es el body del PATCH /api/v1/empleados/{id}/estado.
type CambiarEstadoRequest struct {
	Activo bool `json:"activo"`
}

// CambiarContrasenaRequest es el body del POST /api/v1/empleados/{id}/contrasena/cambiar.
type CambiarContrasenaRequest struct {
	NuevaContrasena string `json:"nueva_contrasena"`
}

// ListarEmpleadosParams agrupa filtros y paginación del listado.
type ListarEmpleadosParams struct {
	Q        string
	TiendaID *uint64
	Activo   *bool
	Page     int
	Limit    int
}

// ListarEmpleadosResponse es la respuesta paginada del GET /api/v1/empleados.
type ListarEmpleadosResponse struct {
	Empleados []EmpleadoResponse `json:"empleados"`
	Total     int                `json:"total"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
}

// EmpleadoResponse es la representación pública del empleado (sin contrasena_hash).
type EmpleadoResponse struct {
	ID                       uint64     `json:"id"`
	Nombre                   string     `json:"nombre"`
	Apellido                 string     `json:"apellido"`
	Usuario                  string     `json:"usuario"`
	Rol                      string     `json:"rol"`
	TiendaID                 *uint64    `json:"tienda_id"`
	TipoDocumento            *string    `json:"tipo_documento,omitempty"`
	NumeroDocumento          *string    `json:"numero_documento,omitempty"`
	Telefono                 *string    `json:"telefono,omitempty"`
	Email                    *string    `json:"email,omitempty"`
	FechaNacimiento          *string    `json:"fecha_nacimiento,omitempty"`
	Activo                   bool       `json:"activo"`
	RequiereCambioContrasena bool       `json:"requiere_cambio_contrasena"`
	CreadoEn                 time.Time  `json:"creado_en"`
	ActualizadoEn            time.Time  `json:"actualizado_en"`
}

// CrearEmpleadoResponse extiende EmpleadoResponse con la contraseña temporal (solo en creación).
type CrearEmpleadoResponse struct {
	EmpleadoResponse
	ContrasenaTemporal string `json:"contrasena_temporal"`
}

// ResetContrasenaResponse es la respuesta del POST /api/v1/empleados/{id}/contrasena.
type ResetContrasenaResponse struct {
	ContrasenaTemporal string `json:"contrasena_temporal"`
}

// errorResponse es el esquema estándar de errores de la API (Constitución §Convenciones API).
type errorResponse struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
	Campo   string `json:"campo,omitempty"`
}

// toResponse convierte un Empleado interno en EmpleadoResponse público.
func toResponse(e Empleado) EmpleadoResponse {
	r := EmpleadoResponse{
		ID:                       e.ID,
		Nombre:                   e.Nombre,
		Apellido:                 e.Apellido,
		Usuario:                  e.Usuario,
		Rol:                      e.Rol,
		TiendaID:                 e.TiendaID,
		TipoDocumento:            e.TipoDocumento,
		NumeroDocumento:          e.NumeroDocumento,
		Telefono:                 e.Telefono,
		Email:                    e.Email,
		Activo:                   e.Activo,
		RequiereCambioContrasena: e.RequiereCambioContrasena,
		CreadoEn:                 e.CreadoEn,
		ActualizadoEn:            e.ActualizadoEn,
	}
	if e.FechaNacimiento != nil {
		s := e.FechaNacimiento.Format("2006-01-02")
		r.FechaNacimiento = &s
	}
	return r
}
