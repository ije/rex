package rex

type contextPanicError struct {
	message string
	code    int
}

func (err *contextPanicError) Error() string {
	return err.message
}
