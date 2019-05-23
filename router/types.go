package router

import (
	"net/http"
	"sync"
)

type nodeType uint8

const (
	root nodeType = iota
	static
	param
	catchAll
)

type node struct {
	lock           sync.RWMutex
	name           string
	nodeType       nodeType
	staticChildren map[string]*node
	wildChild      *node
	validate       Validate
	handle         Handle
}

// Params is a map, as returned by the router.
type Params map[string]string

// Validates is a Validate-map to validate router params.
type Validates map[string]Validate

// Handle is a function that can be registered to a route to handle HTTP requests.
type Handle func(w http.ResponseWriter, r *http.Request, params Params)

// Validate is a function to validate route params
type Validate func(s string) (ok bool)
