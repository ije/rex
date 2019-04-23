package rex

type ctxPanicError struct {
	msg string
}

func (err *ctxPanicError) Error() string {
	return err.msg
}
