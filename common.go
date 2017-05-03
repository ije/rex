package webx

import (
	"fmt"
)

func strf(format string, v ...interface{}) string {
	return fmt.Sprintf(format, v...)
}

func errf(format string, v ...interface{}) error {
	return fmt.Errorf(format, v...)
}
