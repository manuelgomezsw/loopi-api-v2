package proveedores

import "time"

// Proveedor es la entidad de dominio que mapea la tabla `proveedores`.
type Proveedor struct {
	ID               uint64    `json:"id"`
	RazonSocial      string    `json:"razon_social"`
	NIT              string    `json:"nit"`
	NombreContacto   *string   `json:"nombre_contacto"`
	TelefonoContacto *string   `json:"telefono_contacto"`
	EmailContacto    *string   `json:"email_contacto"`
	Activo           bool      `json:"activo"`
	CreadoEn         time.Time `json:"creado_en"`
	ActualizadoEn    time.Time `json:"actualizado_en"`
}

// ProveedorDetalleResponse extiende Proveedor con el conteo de items asignados.
type ProveedorDetalleResponse struct {
	Proveedor
	ItemsAsignados int `json:"items_asignados"`
}

// CrearProveedorRequest es el body de creación de proveedor.
type CrearProveedorRequest struct {
	RazonSocial      string  `json:"razon_social"`
	NIT              string  `json:"nit"`
	NombreContacto   *string `json:"nombre_contacto"`
	TelefonoContacto *string `json:"telefono_contacto"`
	EmailContacto    *string `json:"email_contacto"`
}

// EditarProveedorRequest es el body de edición (todos los campos opcionales).
type EditarProveedorRequest struct {
	RazonSocial      *string `json:"razon_social"`
	NIT              *string `json:"nit"`
	NombreContacto   *string `json:"nombre_contacto"`
	TelefonoContacto *string `json:"telefono_contacto"`
	EmailContacto    *string `json:"email_contacto"`
}

// FiltrosListado agrupa los filtros y paginación del listado.
type FiltrosListado struct {
	Activo   *bool
	Busqueda string
	Page     int
	Limit    int
}

// ListarProveedoresResponse es la respuesta paginada del listado.
type ListarProveedoresResponse struct {
	Proveedores []Proveedor `json:"proveedores"`
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
}

// CambiarEstadoResponse confirma la inactivación o reactivación de un proveedor.
type CambiarEstadoResponse struct {
	ID      uint64 `json:"id"`
	Activo  bool   `json:"activo"`
	Mensaje string `json:"mensaje"`
}

// errorResponse es el esquema estándar de errores de la API.
type errorResponse struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
	Campo   string `json:"campo,omitempty"`
}
