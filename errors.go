package rex

type contextPanicError struct {
	msg string
}

func (err *contextPanicError) Error() string {
	return err.msg
}

type InvalidError struct {
	Code    int
	Message string
}

// Invalid returns a new InvalidError
func Invalid(code int, message string) *InvalidError {
	return &InvalidError{code, message}
}

func (err *InvalidError) Error() string {
	return err.Message
}
