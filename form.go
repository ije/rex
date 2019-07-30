package rex

import (
	"strconv"
)

type FormValue string

func (value FormValue) String() string {
	return string(value)
}

func (value FormValue) Float64() (float64, error) {
	return strconv.ParseFloat(string(value), 64)
}

func (value FormValue) Int64() (int64, error) {
	return strconv.ParseInt(string(value), 10, 64)
}

func (value FormValue) Bool() bool {
	return value == "true" || value == "1" || value == "ok" || value == "yes"
}
