package webx

import (
	"github.com/ije/gox/log"
)

var xs = &XService{
	Log: &log.Logger{},
}

type XService struct {
	App *App
	Log *log.Logger
}

func (xs *XService) clone() *XService {
	return &XService{xs.App, xs.Log}
}

func SetLogger(logger *log.Logger) {
	if logger != nil {
		xs.Log = logger
	}
}
