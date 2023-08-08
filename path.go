package rex

import (
	"strings"

	"github.com/ije/gox/utils"
)

// A Form to handle request path.
type Path struct {
	Params   Params
	raw      string
	segments []string
}

// String returns the path as string
func (path *Path) String() string {
	return "/" + strings.Join(path.segments, "/")
}

func splitPath(path string) []string {
	return strings.Split(utils.CleanPath(path)[1:], "/")
}
