package rex

import (
	"fmt"
	"net/url"
	"strconv"
)

// A Form to handle request url.
type URL struct {
	segments []string
	*url.URL
}

// Segment returns the path segment by the index
func (url *URL) Segment(index int) string {
	if index >= 0 && index < len(url.segments) {
		return url.segments[index]
	}
	return ""
}

// IntSegment returns the path segment as int by the index
func (url *URL) IntSegment(index int) (int64, error) {
	value := url.Segment(index)
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(value, 10, 64)
}

// FloatSegment returns the path segment as float by the index
func (url *URL) FloatSegment(index int) (float64, error) {
	value := url.Segment(index)
	if value == "" {
		return 0.0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(value, 64)
}

// RequireSegment requires a path segment by the index
func (url *URL) RequireSegment(index int) string {
	value := url.Segment(index)
	if value == "" {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d]", index)})
	}
	return value
}

// RequireIntSegment requires a path segment as int by the index
func (url *URL) RequireIntSegment(index int) int64 {
	i, err := url.IntSegment(index)
	if err != nil {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d] as int", index)})
	}
	return i
}

// RequireFloatSegment requires a path segment as float by the index
func (url *URL) RequireFloatSegment(index int) float64 {
	f, err := url.FloatSegment(index)
	if err != nil {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d] as float", index)})
	}
	return f
}
