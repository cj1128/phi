package fasthttpchi

import (
	"github.com/valyala/fasthttp"
)

// Chain returns a Middlewares type from a slice of middleware handlers.
func Chain(middlewares ...Middleware) Middlewares {
	return Middlewares(middlewares)
}

// Handler builds and returns a http.Handler from the chain of middlewares,
// with `h http.Handler` as the final handler.
func (mws Middlewares) Handler(h Handler) Handler {
	return &ChainHandler{mws, h, chain(mws, h)}
}

// HandlerFunc builds and returns a http.Handler from the chain of middlewares,
// with `h http.Handler` as the final handler.
func (mws Middlewares) HandlerFunc(h HandlerFunc) Handler {
	return &ChainHandler{mws, h, chain(mws, h)}
}

// ChainHandler is a http.Handler with support for handler composition and
// execution.
type ChainHandler struct {
	Middlewares Middlewares
	Endpoint    Handler
	chain       Handler
}

func (c *ChainHandler) ServeFastHTTP(ctx *fasthttp.RequestCtx) {
	c.chain.ServeFastHTTP(ctx)
}

// chain builds a http.Handler composed of an inline middleware stack and endpoint
// handler in the order they are passed.
func chain(middlewares Middlewares, endpoint Handler) Handler {
	// Return ahead of time if there aren't any middlewares for the chain
	if len(middlewares) == 0 {
		return endpoint
	}

	// Wrap the end handler with the middleware chain
	h := middlewares[len(middlewares)-1](endpoint.ServeFastHTTP)
	for i := len(middlewares) - 2; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}
