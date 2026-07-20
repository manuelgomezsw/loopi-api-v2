package completar

import (
	"github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"
	"time"
)

// ConfirmarRequest es el request para confirmar la finalización de un conteo
type ConfirmarRequest struct {
	Confirmar bool `json:"confirmar" binding:"required"`
}

// ConfirmarResponse es la respuesta al confirmar un conteo
type ConfirmarResponse struct {
	ID             int64                    `json:"id"`
	TiendaID       int64                    `json:"tienda_id"`
	Estado         core.Estado              `json:"estado"`
	CompletadoEn   time.Time                `json:"completado_en"`
	ItemsAjustados int                      `json:"items_ajustados"`
	Resumen        *ResumenConfirmacion     `json:"resumen"`
	Timestamp      time.Time                `json:"timestamp"`
}

// ResumenConfirmacion es el resumen de la confirmación de conteo
type ResumenConfirmacion struct {
	TotalItems          int     `json:"total_items"`
	ItemsCorrectos      int     `json:"items_correctos"`
	ItemsFaltantes      int     `json:"items_faltantes"`
	ItemsExceso         int     `json:"items_exceso"`
	DiferenciaNegativa  float64 `json:"diferencia_negativa"`
	DiferenciaPositiva  float64 `json:"diferencia_positiva"`
	PorcentajeVariacion float64 `json:"porcentaje_variacion"`
}

// ItemAjuste representa el ajuste de stock realizado para cada item
type ItemAjuste struct {
	ItemID           int64   `json:"item_id"`
	ItemCodigo       string  `json:"item_codigo"`
	ItemDescripcion  string  `json:"item_descripcion"`
	ValorEsperado    float64 `json:"valor_esperado"`
	ValorReal        float64 `json:"valor_real"`
	Diferencia       float64 `json:"diferencia"`
	StockAnterior    float64 `json:"stock_anterior"`
	StockNuevo       float64 `json:"stock_nuevo"`
	Unidad           string  `json:"unidad"`
}
