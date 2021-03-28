package rex

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ije/gox/utils"
)

// A Form to handle request path.
type Path struct {
	segments []string
}

// String returns the path as string
func (path *Path) String() string {
	return "/" + strings.Join(path.segments, "/")
}

// Len returns the path segments
func (path *Path) Segments() []string {
	return path.segments
}

// Update sets a new path
func (path *Path) Update(pathname string) {
	path.segments = strings.Split(utils.CleanPath(pathname), "/")[1:]
}

// Segment returns the path segment by the index
func (path *Path) GetSegment(index int) string {
	if index >= 0 && index < len(path.segments) {
		return path.segments[index]
	}
	return ""
}

// IntSegment returns the path segment as int by the index
func (path *Path) GetIntSegment(index int) (int64, error) {
	value := path.GetSegment(index)
	if value == "" {
		return 0, strconv.ErrSyntax
	}
	return strconv.ParseInt(value, 10, 64)
}

// FloatSegment returns the path segment as float by the index
func (path *Path) GetFloatSegment(index int) (float64, error) {
	value := path.GetSegment(index)
	if value == "" {
		return 0.0, strconv.ErrSyntax
	}
	return strconv.ParseFloat(value, 64)
}

// RequireSegment requires a path segment by the index
func (path *Path) RequireSegment(index int) string {
	value := path.GetSegment(index)
	if value == "" {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d]", index)})
	}
	return value
}

// RequireIntSegment requires a path segment as int by the index
func (path *Path) RequireIntSegment(index int) int64 {
	i, err := path.GetIntSegment(index)
	if err != nil {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d] as int", index)})
	}
	return i
}

// RequireFloatSegment requires a path segment as float by the index
func (path *Path) RequireFloatSegment(index int) float64 {
	f, err := path.GetFloatSegment(index)
	if err != nil {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d] as float", index)})
	}
	return f
}
