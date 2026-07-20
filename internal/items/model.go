package items

import "time"

// Item es la entidad de dominio que mapea la tabla `items`.
type Item struct {
	ID                   uint64    `json:"id"`
	Codigo               string    `json:"codigo"`
	Nombre               string    `json:"nombre"`
	Tipo                 string    `json:"tipo"`
	SubcategoriaID       uint64    `json:"subcategoria_id"`
	ProveedorID          *uint64   `json:"proveedor_id"`
	UnidadMedidaID       uint64    `json:"unidad_medida_id"`
	CostoUnitario        *int      `json:"costo_unitario"`
	FrecuenciaInventario string    `json:"frecuencia_inventario"`
	StockSeguridad       string    `json:"stock_seguridad"`
	TiempoEntregaDias    *uint16   `json:"tiempo_entrega_dias"`
	Activo               bool      `json:"activo"`
	CreadoPor            uint64    `json:"creado_por"`
	CreadoEn             time.Time `json:"creado_en"`
	ActualizadoPor       uint64    `json:"actualizado_por"`
	ActualizadoEn        time.Time `json:"actualizado_en"`
}

// ItemConNombres extiende Item con los nombres resueltos de sus FKs. Usado en el listado.
type ItemConNombres struct {
	Item
	SubcategoriaNombre  string  `json:"subcategoria_nombre"`
	ProveedorNombre     *string `json:"proveedor_nombre,omitempty"`
	UnidadMedidaSimbolo string  `json:"unidad_medida_simbolo"`
}

// ItemDetalleResponse extiende ItemConNombres con el flag de uso, exclusivo del detalle
// y de las respuestas de creación/edición. EstaEnUso no se completa a nivel de repositorio
// — es responsabilidad del service.
type ItemDetalleResponse struct {
	ItemConNombres
	EstaEnUso bool `json:"esta_en_uso"`
}

// CrearItemRequest es el body de creación de item.
type CrearItemRequest struct {
	Codigo               string  `json:"codigo"`
	Nombre               string  `json:"nombre"`
	Tipo                 string  `json:"tipo"`
	SubcategoriaID       uint64  `json:"subcategoria_id"`
	ProveedorID          *uint64 `json:"proveedor_id"`
	UnidadMedidaID       uint64  `json:"unidad_medida_id"`
	CostoUnitario        *int    `json:"costo_unitario"`
	FrecuenciaInventario string  `json:"frecuencia_inventario"`
	StockSeguridad       string  `json:"stock_seguridad"`
	TiempoEntregaDias    *uint16 `json:"tiempo_entrega_dias"`
}

// EditarItemRequest es el body de edición de item.
type EditarItemRequest struct {
	Codigo                string  `json:"codigo"`
	Nombre                string  `json:"nombre"`
	SubcategoriaID        uint64  `json:"subcategoria_id"`
	ProveedorID           *uint64 `json:"proveedor_id"`
	UnidadMedidaID        uint64  `json:"unidad_medida_id"`
	CostoUnitario         *int    `json:"costo_unitario"`
	FrecuenciaInventario  string  `json:"frecuencia_inventario"`
	StockSeguridad        string  `json:"stock_seguridad"`
	TiempoEntregaDias     *uint16 `json:"tiempo_entrega_dias"`
	ConfirmarCambioUnidad bool    `json:"confirmar_cambio_unidad"`
}

// FiltrosListado agrupa los filtros y paginación del listado de items.
type FiltrosListado struct {
	Tipo       string
	Frecuencia string
	Activo     *bool
	Pagina     int
	PorPagina  int
}

// ListarItemsResponse es la respuesta paginada del listado. No incluye esta_en_uso
// (exclusivo del detalle individual, según contracts/api.md §1).
type ListarItemsResponse struct {
	Items        []ItemConNombres `json:"items"`
	Total        int              `json:"total"`
	Pagina       int              `json:"pagina"`
	TotalPaginas int              `json:"total_paginas"`
}

// CambiarEstadoResponse confirma la inactivación o reactivación de un item.
type CambiarEstadoResponse struct {
	ID             uint64    `json:"id"`
	Codigo         string    `json:"codigo"`
	Nombre         string    `json:"nombre"`
	Activo         bool      `json:"activo"`
	ActualizadoPor uint64    `json:"actualizado_por"`
	ActualizadoEn  time.Time `json:"actualizado_en"`
}

// ItemCostoTienda es la entidad de dominio que mapea la tabla `items_costos_tienda`.
// Usada como respuesta completa del registro de un nuevo costo (POST costos_tienda).
type ItemCostoTienda struct {
	ID            uint64    `json:"id"`
	ItemID        uint64    `json:"item_id"`
	TiendaID      uint64    `json:"tienda_id"`
	CostoUnitario int       `json:"costo_unitario"`
	VigenteDesde  time.Time `json:"vigente_desde"`
	CreadoPor     uint64    `json:"creado_por"`
	CreadoEn      time.Time `json:"creado_en"`
}

// RegistrarCostoTiendaRequest es el body de registro de costo por tienda.
type RegistrarCostoTiendaRequest struct {
	TiendaID      uint64 `json:"tienda_id"`
	CostoUnitario int    `json:"costo_unitario"`
}

// HistorialEntry es una entrada individual del historial de costos de una tienda,
// anidada dentro de CostoPorTienda (sin item_id/tienda_id redundantes).
type HistorialEntry struct {
	ID            uint64    `json:"id"`
	CostoUnitario int       `json:"costo_unitario"`
	VigenteDesde  time.Time `json:"vigente_desde"`
	CreadoPor     uint64    `json:"creado_por"`
	CreadoEn      time.Time `json:"creado_en"`
}

// CostoPorTienda agrupa el historial de costos de una tienda con el costo vigente.
type CostoPorTienda struct {
	TiendaID     uint64           `json:"tienda_id"`
	TiendaNombre string           `json:"tienda_nombre"`
	CostoVigente int              `json:"costo_vigente"`
	Historial    []HistorialEntry `json:"historial"`
}

// HistorialCostosResponse es la respuesta del historial de costos por tienda de un item.
type HistorialCostosResponse struct {
	ItemID          uint64           `json:"item_id"`
	CostoGlobal     *int             `json:"costo_global"`
	CostosPorTienda []CostoPorTienda `json:"costos_por_tienda"`
}

// errorResponse es el esquema estándar de errores de la API.
type errorResponse struct {
	Error   string `json:"error"`
	Mensaje string `json:"mensaje"`
	Campo   string `json:"campo,omitempty"`
}
