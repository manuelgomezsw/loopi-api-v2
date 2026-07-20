package core

import (
	"time"
)

// Tipo enum para categorías de conteo
type Tipo string

const (
	TipoDiario   Tipo = "diario"
	TipoSemanal  Tipo = "semanal"
	TipoMensual  Tipo = "mensual"
	TipoInicial  Tipo = "inicial"
)

// Horario enum para franjas diarias
type Horario string

const (
	HorarioApertura  Horario = "apertura"
	HorarioMediodia  Horario = "mediodia"
	HoriarioCierre   Horario = "cierre"
)

// Estado enum para ciclo de vida del inventario
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

// TipoMovimiento enum para auditoría de stock
type TipoMovimiento string

const (
	TipoMovimientoCompra       TipoMovimiento = "compra"
	TipoMovimientoMerma        TipoMovimiento = "merma"
	TipoMovimientoVentaBatch   TipoMovimiento = "venta_batch"
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
