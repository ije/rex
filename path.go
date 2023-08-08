package rex

// A Form to handle request path.
type Path struct {
	Params Params
	raw    string
}

// String returns the path as string
func (path *Path) String() string {
	return path.raw
}
