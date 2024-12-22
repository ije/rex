package rex

import (
	"net/http"
	"sync"
)

var (
	contextPool = sync.Pool{
		New: func() any {
			return &Context{}
		},
	}
	writerPool = sync.Pool{
		New: func() any {
			return &rexWriter{}
		},
	}
)

// newContext returns a new Context from the pool.
func newContext(r *http.Request) (ctx *Context, recycle func()) {
	ctx = contextPool.Get().(*Context)
	recycle = func() {
		contextPool.Put(ctx)
	}
	ctx.R = r
	ctx.basicAuthUser = ""
	ctx.aclUser = nil
	ctx.session = nil
	ctx.sessionPool = defaultSessionPool
	ctx.sessionIdHandler = defaultSessionIdHandler
	ctx.logger = defaultLogger
	ctx.accessLogger = nil
	ctx.compress = false
	return
}

// newWriter returns a new Writer from the pool.
func newWriter(ctx *Context, w http.ResponseWriter) (wr *rexWriter, recycle func()) {
	wr = writerPool.Get().(*rexWriter)
	recycle = func() {
		writerPool.Put(wr)
	}
	wr.ctx = ctx
	wr.rawWriter = w
	wr.code = 200
	wr.headerSent = false
	wr.writeN = 0
	wr.zWriter = nil
	return
}
