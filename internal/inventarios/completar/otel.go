package completar

// Constantes OTel para spans del módulo completar conteo
const (
	// Span principal de confirmación
	SpanConfirmar = "inventario.completar.confirmar"

	// Span de validación de completitud
	SpanValidarCompletitud = "inventario.completar.validar_completitud"

	// Span de actualización de stock
	SpanActualizarStock = "inventario.completar.actualizar_stock"

	// Atributos OTel
	AttrResultado      = "resultado"
	AttrTiendaID       = "tienda_id"
	AttrInventarioID   = "inventario_id"
	AttrItemsCount     = "items_count"
	AttrItemsIncompletos = "items_incompletos"
	AttrItemsAjustados = "items_ajustados"
	AttrDiferenciaFaltantes = "diferencias_faltantes"
	AttrDiferenciaExcesos = "excesos"
	AttrDiferenciaCorrectos = "correctas"
)
