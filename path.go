package rex

import (
	"fmt"
)

// A Form to handle request path.
type Path struct {
	raw      string
	segments []string
}

// String returns the path as string
func (path *Path) String() string {
	return path.raw
}

// Len returns the path segments
func (path *Path) Segments() []string {
	segments := make([]string, len(path.segments))
	copy(segments, path.segments)
	return segments
}

// Segment returns the path segment by the index
func (path *Path) GetSegment(index int) string {
	if index >= 0 && index < len(path.segments) {
		return path.segments[index]
	}
	return ""
}

// RequireSegment requires a path segment by the index
func (path *Path) RequireSegment(index int) string {
	value := path.GetSegment(index)
	if value == "" {
		panic(&recoverError{400, fmt.Sprintf("require path segment[%d]", index)})
	}
	return value
}
