package completar

// Error personalizado para el servicio de completar conteo
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
	ErrConteoIncompleto      = NewError("CONTEO_INCOMPLETO", "Existen items sin valor real registrado")
	ErrConteoVacio           = NewError("CONTEO_VACIO", "El conteo no tiene items registrados")
	ErrConteoNoEnProgreso    = NewError("CONTEO_NO_EN_PROGRESO", "El conteo no está en estado 'en_progreso'")
	ErrConteoYaCompletado    = NewError("CONTEO_YA_COMPLETADO", "El conteo ya ha sido completado")
	ErrConteoNoEncontrado    = NewError("CONTEO_NO_ENCONTRADO", "Conteo no encontrado")
	ErrPermisoInsuficiente   = NewError("PERMISO_INSUFICIENTE", "Usuario no tiene permisos para completar este conteo")
	ErrValidacionFallo       = NewError("VALIDATION_ERROR", "Validación falló")
	ErrTransaccionFallo      = NewError("TRANSACTION_ERROR", "Error en transacción de base de datos")
)
