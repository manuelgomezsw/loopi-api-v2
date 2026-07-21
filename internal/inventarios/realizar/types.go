package realizar

import (
	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
	"time"
)

// RegistrarValorRequest es el request para registrar un valor en un item del conteo
type RegistrarValorRequest struct {
	ValorReal float64 `json:"valor_real" binding:"required,min=0"`
}

// RegistrarValorResponse es la respuesta al registrar un valor
type RegistrarValorResponse struct {
	Success              bool      `json:"success"`
	ItemID               int64     `json:"item_id"`
	ItemCodigo           string    `json:"item_codigo"`
	ItemDescripcion      string    `json:"item_descripcion"`
	ValorEsperado        float64   `json:"valor_esperado"`
	ValorReal            float64   `json:"valor_real"`
	Diferencia           float64   `json:"diferencia"`
	DiferenciaPorcentaje float64   `json:"diferencia_porcentaje"`
	Unidad               string    `json:"unidad"`
	Timestamp            time.Time `json:"timestamp"`
}

// GetDetallesResponse es la respuesta al cargar items para precarga
type GetDetallesResponse struct {
	InventarioID int64             `json:"inventario_id"`
	TiendaID     int64             `json:"tienda_id"`
	Estado       core.Estado       `json:"estado"`
	Items        []ItemDetalle     `json:"items"`
	Resumen      ResumenProgreso   `json:"resumen"`
}

// ItemDetalle es el detalle de un item en el conteo
type ItemDetalle struct {
	ItemID         int64      `json:"item_id"`
	ItemCodigo     string     `json:"item_codigo"`
	ItemDescripcion string    `json:"item_descripcion"`
	UnidadMedidaID int64      `json:"unidad_medida_id,omitempty"`
	Unidad         string     `json:"unidad"`
	ValorEsperado  float64    `json:"valor_esperado"`
	ValorReal      *float64   `json:"valor_real"`
	Diferencia     *float64   `json:"diferencia"`
	Completado     bool       `json:"completado"`
}

// ResumenProgreso es el resumen del progreso del conteo
type ResumenProgreso struct {
	TotalItems        int     `json:"total_items"`
	Completados       int     `json:"completados"`
	Pendientes        int     `json:"pendientes"`
	PorcentajeProgreso float64 `json:"porcentaje_progreso"`
}
