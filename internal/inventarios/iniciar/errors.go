package iniciar

// Error personalizado para el servicio de iniciar
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
