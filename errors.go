package wsx

type initSessionError struct {
	msg string
}

func (err *initSessionError) Error() string {
	return err.msg
}
