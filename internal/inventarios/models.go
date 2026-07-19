package inventarios

import (
	"time"
)

// Tipo enum
type Tipo string

const (
	TipoDiario   Tipo = "diario"
	TipoSemanal  Tipo = "semanal"
	TipoMensual  Tipo = "mensual"
	TipoInicial  Tipo = "inicial"
)

// Horario enum
type Horario string

const (
	HorarioApertura  Horario = "apertura"
	HorarioMediodia  Horario = "mediodia"
	HoriarioCierre   Horario = "cierre"
)

// Estado enum
type Estado string

const (
	EstadoEnProgreso Estado = "en_progreso"
	EstadoCompletado Estado = "completado"
)

// Inventario representa un conteo físico de inventario
type Inventario struct {
	ID              int64      `db:"id" json:"id"`
	TiendaID        int64      `db:"tienda_id" json:"tienda_id"`
	Fecha           time.Time  `db:"fecha" json:"fecha"`
	Tipo            Tipo       `db:"tipo" json:"tipo"`
	Horario         *Horario   `db:"horario" json:"horario"`
	HorarioNorm     string     `db:"horario_norm" json:"-"`
	Estado          Estado     `db:"estado" json:"estado"`
	ResponsableID   int64      `db:"responsable_id" json:"responsable_id"`
	IniciadoEn      time.Time  `db:"iniciado_en" json:"iniciado_en"`
	CompletadoEn    *time.Time `db:"completado_en" json:"completado_en"`
	CreadoEn        time.Time  `db:"creado_en" json:"creado_en"`
	ActualizadoEn   time.Time  `db:"actualizado_en" json:"actualizado_en"`

	// Relaciones (para respuestas)
	Items []DetalleInventario `json:"items,omitempty"`
}

// DetalleInventario representa una línea del conteo
type DetalleInventario struct {
	ID                        int64       `db:"id" json:"id"`
	InventarioID              int64       `db:"inventario_id" json:"inventario_id"`
	ItemID                    int64       `db:"item_id" json:"item_id"`
	Nombre                    string      `db:"nombre" json:"nombre"`
	UnidadMedidaID            interface{} `db:"unidad_medida_id" json:"unidad_medida_id,omitempty"`
	InventarioReferenciaID    *int64      `db:"inventario_referencia_id" json:"inventario_referencia_id"`
	ValorEsperado             float64     `db:"valor_esperado" json:"valor_esperado"`
	ValorReal                 *float64    `db:"valor_real" json:"valor_real"`
	Diferencia                *float64    `db:"diferencia" json:"diferencia"`
	CreadoEn                  time.Time   `db:"creado_en" json:"creado_en"`
	ActualizadoEn             time.Time   `db:"actualizado_en" json:"actualizado_en"`
}

// Request DTOs

// CreateInventarioReq es el request para iniciar un conteo
type CreateInventarioReq struct {
	TiendaID   int64   `json:"tienda_id" binding:"required"`
	Tipo       Tipo    `json:"tipo" binding:"required"`
	Horario    *Horario `json:"horario"`
}

// PatchItemValorReq es el request para registrar un valor real
type PatchItemValorReq struct {
	ValorReal float64 `json:"valor_real" binding:"required"`
}

// Response DTOs

// InventarioResp es la respuesta para un inventario con detalles
type InventarioResp struct {
	ID            int64              `json:"id"`
	TiendaID      int64              `json:"tienda_id"`
	Fecha         time.Time          `json:"fecha"`
	Tipo          Tipo               `json:"tipo"`
	Horario       *Horario           `json:"horario"`
	Estado        Estado             `json:"estado"`
	ResponsableID int64              `json:"responsable_id"`
	IniciadoEn    time.Time          `json:"iniciado_en"`
	CompletadoEn  *time.Time         `json:"completado_en"`
	Items         []ItemDetailResp   `json:"items"`
}

// ItemDetailResp es la respuesta para un detalle de item
type ItemDetailResp struct {
	ID                     int64    `json:"id"`
	ItemID                 int64    `json:"item_id"`
	Nombre                 string   `json:"nombre"`
	UnidadMedidaID         interface{} `json:"unidad_medida_id,omitempty"`
	ValorEsperado          float64  `json:"valor_esperado"`
	ValorReal              *float64 `json:"valor_real"`
	Diferencia             *float64 `json:"diferencia"`
}

// SugerenciaResp es la respuesta para la sugerencia de tipo/horario
type SugerenciaResp struct {
	Tipo    Tipo    `json:"tipo"`
	Horario Horario `json:"horario"`
}

// HistorialResp es la respuesta para el listado de inventarios
type HistorialResp struct {
	Inventarios []InventarioResp `json:"inventarios"`
	Total       int64            `json:"total"`
	Pagina      int              `json:"pagina"`
	TotalPaginas int              `json:"total_paginas"`
}

// ErrorResp es la respuesta de error estándar
type ErrorResp struct {
	Error      string        `json:"error"`
	Mensaje    string        `json:"mensaje"`
	Campo      *string       `json:"campo,omitempty"`
	Detalles   interface{}   `json:"detalles,omitempty"`
}

// TipoMovimiento enum
type TipoMovimiento string

const (
	TipoMovimientoCompra      TipoMovimiento = "compra"
	TipoMovimientoMerma       TipoMovimiento = "merma"
	TipoMovimientoVentaBatch  TipoMovimiento = "venta_batch"
	TipoMovimientoAjusteConteo TipoMovimiento = "ajuste_conteo"
)

// StockMovimiento representa un movimiento de stock registrado en auditoría
type StockMovimiento struct {
	ID              int64          `db:"id" json:"id"`
	TiendaID        int64          `db:"tienda_id" json:"tienda_id"`
	ItemID          int64          `db:"item_id" json:"item_id"`
	TipoMovimiento  TipoMovimiento `db:"tipo_movimiento" json:"tipo_movimiento"`
	CantidadAntes   float64        `db:"cantidad_antes" json:"cantidad_antes"`
	CantidadDespues float64        `db:"cantidad_despues" json:"cantidad_despues"`
	CantidadDelta   float64        `db:"cantidad_delta" json:"cantidad_delta"`
	ReferenciaID    *int64         `db:"referencia_id" json:"referencia_id"`
	ReferenciaTipo  *string        `db:"referencia_tipo" json:"referencia_tipo"`
	UsuarioID       int64          `db:"usuario_id" json:"usuario_id"`
	Motivo          *string        `db:"motivo" json:"motivo"`
	CreadoEn        time.Time      `db:"creado_en" json:"creado_en"`
}

// StockActual representa un snapshot del stock al iniciar conteo
type StockActual struct {
	ID            int64     `db:"id" json:"id"`
	TiendaID      int64     `db:"tienda_id" json:"tienda_id"`
	ItemID        int64     `db:"item_id" json:"item_id"`
	InventarioID  int64     `db:"inventario_id" json:"inventario_id"`
	ValorSnapshot float64   `db:"valor_snapshot" json:"valor_snapshot"`
	TomadoEn      time.Time `db:"tomado_en" json:"tomado_en"`
	CreadoEn      time.Time `db:"creado_en" json:"creado_en"`
}

// EstadoInventarioResp es la respuesta para verificar si hay conteo activo
type EstadoInventarioResp struct {
	Activo      bool             `json:"activo"`
	Inventario  *InventarioResp  `json:"inventario,omitempty"`
}
