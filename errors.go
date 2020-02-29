package rex

// InvalidError is an error with code
type InvalidError struct {
	Code    int
	Message string
}

// Invalid returns a new InvalidError
func Invalid(code int, message string) *InvalidError {
	return &InvalidError{code, message}
}

// Error implements the error type
func (err *InvalidError) Error() string {
	return err.Message
}

type contextPanicError struct {
	code    int
	message string
}

func (err *contextPanicError) Error() string {
	return err.message
}
