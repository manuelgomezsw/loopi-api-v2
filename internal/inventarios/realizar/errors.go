package realizar

// Error personalizado para el servicio de realizar
type Error struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *Error) Error() string {
	return e.Message
}

func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

func NewErrorWithDetails(code, message string, details map[string]interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Errores específicos del módulo
var (
	ErrValorNegativo       = NewError("INVALID_VALOR", "valor_real debe ser >= 0")
	ErrConteoNoProgreso    = NewError("CONTEO_NO_EN_PROGRESO", "El conteo no está en estado 'en_progreso'")
	ErrItemNoEncontrado    = NewError("ITEM_NO_ENCONTRADO", "item no pertenece a este conteo")
	ErrConteoNoEncontrado  = NewError("CONTEO_NO_ENCONTRADO", "conteo no encontrado")
	ErrValidacionFallo     = NewError("VALIDATION_ERROR", "validación falló")
)
