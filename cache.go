package rex

import (
	"io"
)

type CacheStorage interface {
	Get(key string) (*io.ReadCloser, bool, error)
	Set(key string, r *io.Reader) error
}

// Cache returns a middleware that caches the response.
func Cache(storage CacheStorage) Handle {
	return func(ctx *Context) any {
		ctx.cache = storage
		return next
	}
}
