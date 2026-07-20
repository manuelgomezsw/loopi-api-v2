package unidades_medida

import "time"

// UnidadMedida representa una unidad de medida del catálogo compartido.
type UnidadMedida struct {
	ID               uint64    `json:"id"`
	Codigo           string    `json:"codigo"`
	Nombre           string    `json:"nombre"`
	TipoMedida       string    `json:"tipo_medida"`
	FactorConversion float64   `json:"factor_conversion"`
	UnidadBase       bool      `json:"unidad_base"`
	Activo           bool      `json:"activo"`
	CreadoEn         time.Time `json:"creado_en"`
	ActualizadoEn    time.Time `json:"actualizado_en"`
}

// DetalleUMResponse extiende UnidadMedida con el conteo de items que la usan como canónica.
type DetalleUMResponse struct {
	UnidadMedida
	ItemsConUnidadCanonica int `json:"items_con_unidad_canonica"`
}

// CrearUMRequest es el body de creación de unidad de medida.
type CrearUMRequest struct {
	Codigo           string  `json:"codigo"`
	Nombre           string  `json:"nombre"`
	TipoMedida       string  `json:"tipo_medida"`
	FactorConversion float64 `json:"factor_conversion"`
}

// EditarUMRequest es el body de edición (todos los campos opcionales).
type EditarUMRequest struct {
	Nombre           *string  `json:"nombre"`
	FactorConversion *float64 `json:"factor_conversion"`
}

// ListarUMParams agrupa los filtros y paginación del listado.
type ListarUMParams struct {
	Tipo   string
	Activo *bool
	Page   int
	Limit  int
}

// ListarUMResponse es la respuesta paginada del listado.
type ListarUMResponse struct {
	UnidadesMedida []UnidadMedida `json:"unidades_medida"`
	Total          int            `json:"total"`
	Page           int            `json:"page"`
	Limit          int            `json:"limit"`
}

// ImpactoResponse informa cuántos items activos usan esta unidad como canónica.
type ImpactoResponse struct {
	UnidadID               uint64  `json:"unidad_id"`
	ItemsConUnidadCanonica int     `json:"items_con_unidad_canonica"`
	Advertencia            *string `json:"advertencia"`
}

// InactivarResponse confirma la inactivación de una unidad.
type InactivarResponse struct {
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
