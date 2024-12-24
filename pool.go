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
func newContext(r *http.Request) (ctx *Context) {
	ctx = contextPool.Get().(*Context)
	ctx.R = r
	ctx.sessionPool = defaultSessionPool
	ctx.sessionIdHandler = defaultSessionIdHandler
	ctx.logger = defaultLogger
	return
}

// recycleContext puts a Context back to the pool.
func recycleContext(ctx *Context) {
	ctx.R = nil
	ctx.W = nil
	ctx.Header = nil
	ctx.basicAuthUser = ""
	ctx.aclUser = nil
	ctx.session = nil
	ctx.sessionPool = nil
	ctx.sessionIdHandler = nil
	ctx.logger = nil
	ctx.accessLogger = nil
	ctx.compress = false
	contextPool.Put(ctx)
}

// newWriter returns a new Writer from the pool.
func newWriter(ctx *Context, w http.ResponseWriter) (wr *rexWriter) {
	wr = writerPool.Get().(*rexWriter)
	wr.ctx = ctx
	wr.rawWriter = w
	wr.code = 200
	return
}

// recycleWriter puts a Writer back to the pool.
func recycleWriter(wr *rexWriter) {
	wr.ctx = nil
	wr.rawWriter = nil
	wr.isHeaderSent = false
	wr.writeN = 0
	wr.zWriter = nil
	writerPool.Put(wr)
}
