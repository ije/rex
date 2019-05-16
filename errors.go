package rex

type contextPanicError struct {
	msg string
}

func (err *contextPanicError) Error() string {
	return err.msg
}
