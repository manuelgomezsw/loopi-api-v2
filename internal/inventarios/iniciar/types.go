package iniciar

import "github.com/manuelgomezsw/loopi-api-v2/internal/inventarios/core"

// Tipos de request/response compartidos con handler

// CreateInventarioReq es el request para iniciar un conteo
type CreateInventarioReq struct {
	TiendaID int64         `json:"tienda_id" binding:"required"`
	Tipo     core.Tipo     `json:"tipo" binding:"required"`
	Horario  *core.Horario `json:"horario"`
}

// InventarioResp es la respuesta para un inventario iniciado
type InventarioResp struct {
	ID            int64            `json:"id"`
	TiendaID      int64            `json:"tienda_id"`
	Tipo          core.Tipo        `json:"tipo"`
	Horario       *core.Horario    `json:"horario"`
	Estado        core.Estado      `json:"estado"`
	ResponsableID int64            `json:"responsable_id"`
	IniciadoEn    string           `json:"iniciado_en"`
	Items         []ItemDetailResp `json:"items"`
}

// ItemDetailResp detalle de item en respuesta
type ItemDetailResp struct {
	ID            int64    `json:"id"`
	ItemID        int64    `json:"item_id"`
	Nombre        string   `json:"nombre"`
	ValorEsperado float64  `json:"valor_esperado"`
	ValorReal     *float64 `json:"valor_real"`
}

// SugerenciaResp es la respuesta para la sugerencia de tipo/horario
type SugerenciaResp struct {
	Tipo    core.Tipo    `json:"tipo"`
	Horario core.Horario `json:"horario"`
}
