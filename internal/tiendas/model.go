package tiendas

import "time"

// Tienda representa un punto de venta físico.
type Tienda struct {
	ID             uint64    `db:"id"`
	Codigo         string    `db:"codigo"`
	Nombre         string    `db:"nombre"`
	Direccion      string    `db:"direccion"`
	Ciudad         string    `db:"ciudad"`
	Telefono       string    `db:"telefono"`
	Activo         bool      `db:"activo"`
	CreadoPor      uint64    `db:"creado_por"`
	CreadoEn       time.Time `db:"creado_en"`
	ActualizadoPor uint64    `db:"actualizado_por"`
	ActualizadoEn  time.Time `db:"actualizado_en"`
}

// TiendaRequest es el body de creación.
type TiendaRequest struct {
	Codigo    string `json:"codigo"`
	Nombre    string `json:"nombre"`
	Direccion string `json:"direccion"`
	Ciudad    string `json:"ciudad"`
	Telefono  string `json:"telefono"`
}

// TiendaUpdateRequest es el body de edición (campo Codigo ignorado).
type TiendaUpdateRequest struct {
	Nombre    string `json:"nombre"`
	Direccion string `json:"direccion"`
	Ciudad    string `json:"ciudad"`
	Telefono  string `json:"telefono"`
}

// TiendaResponse es la representación pública de la entidad.
type TiendaResponse struct {
	ID             uint64    `json:"id"`
	Codigo         string    `json:"codigo"`
	Nombre         string    `json:"nombre"`
	Direccion      string    `json:"direccion"`
	Ciudad         string    `json:"ciudad"`
	Telefono       string    `json:"telefono"`
	Activo         bool      `json:"activo"`
	CreadoPor      uint64    `json:"creado_por"`
	CreadoEn       time.Time `json:"creado_en"`
	ActualizadoPor uint64    `json:"actualizado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

// ListaTiendasResponse es la respuesta paginada del listado.
type ListaTiendasResponse struct {
	Datos  []TiendaResponse `json:"datos"`
	Total  int              `json:"total"`
	Pagina int              `json:"pagina"`
	Limite int              `json:"limite"`
}

// errorResponse es el esquema estándar de errores de la API.
type errorResponse struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
	Campo   string `json:"campo,omitempty"`
}

// toResponse convierte una Tienda en TiendaResponse.
func toResponse(t Tienda) TiendaResponse {
	return TiendaResponse(t)
}
