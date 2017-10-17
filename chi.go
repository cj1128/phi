//
// Package chi is a small, idiomatic and composable router for building HTTP services.
//
// chi requires Go 1.7 or newer.
//
// Example:
//  package main
//
//  import (
//  	"net/http"
//
//  	"github.com/go-chi/chi"
//  	"github.com/go-chi/chi/middleware"
//  )
//
//  func main() {
//  	r := chi.NewRouter()
//  	r.Use(middleware.Logger)
//  	r.Use(middleware.Recoverer)
//
//  	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
//  		w.Write([]byte("root."))
//  	})
//
//  	http.ListenAndServe(":3333", r)
//  }
//
// See github.com/go-chi/chi/_examples/ for more in-depth examples.
//
package fasthttpchi

import (
	"github.com/valyala/fasthttp"
)

type Handler interface {
	ServeFastHTTP(ctx *fasthttp.RequestCtx)
}

type HandlerFunc func(ctx *fasthttp.RequestCtx)

func (fn HandlerFunc) ServeFastHTTP(ctx *fasthttp.RequestCtx) {
	fn(ctx)
}

type Middleware func(HandlerFunc) HandlerFunc

// Middlewares type is a slice of standard middleware handlers with methods
// to compose middleware chains and http.Handler's.
// type Middlewares []func(Handler) Handler
type Middlewares []Middleware

// NewRouter returns a new Mux object that implements the Router interface.
func NewRouter() *Mux {
	return NewMux()
}

// Router consisting of the core routing methods used by chi's Mux,
// using only the standard net/http.
type Router interface {
	Handler
	Routes

	// Use appends one of more middlewares onto the Router stack.
	Use(middlewares ...Middleware)

	// With adds inline middlewares for an endpoint handler.
	With(middlewares ...Middleware) Router

	// Group adds a new inline-Router along the current routing
	// path, with a fresh middleware stack for the inline-Router.
	Group(fn func(r Router))

	// Route mounts a sub-Router along a `pattern`` string.
	Route(pattern string, fn func(r Router))

	// Mount attaches another http.Handler along ./pattern/*
	Mount(pattern string, h Handler)

	// Handle and HandleFunc adds routes for `pattern` that matches
	// all HTTP methods.
	Handle(pattern string, h HandlerFunc)

	// Method and MethodFunc adds routes for `pattern` that matches
	// the `method` HTTP method.
	Method(method, pattern string, h HandlerFunc)

	// HTTP-method routing along `pattern`
	Connect(pattern string, h HandlerFunc)
	Delete(pattern string, h HandlerFunc)
	Get(pattern string, h HandlerFunc)
	Head(pattern string, h HandlerFunc)
	Options(pattern string, h HandlerFunc)
	Patch(pattern string, h HandlerFunc)
	Post(pattern string, h HandlerFunc)
	Put(pattern string, h HandlerFunc)
	Trace(pattern string, h HandlerFunc)

	// NotFound defines a handler to respond whenever a route could
	// not be found.
	NotFound(h HandlerFunc)

	// MethodNotAllowed defines a handler to respond whenever a method is
	// not allowed.
	MethodNotAllowed(h HandlerFunc)
}

// Routes interface adds two methods for router traversal, which is also
// used by the `docgen` subpackage to generation documentation for Routers.
type Routes interface {
	// Routes returns the routing tree in an easily traversable structure.
	Routes() []Route

	// Middlewares returns the list of middlewares in use by the router.
	Middlewares() Middlewares

	// Match searches the routing tree for a handler that matches
	// the method/path - similar to routing a http request, but without
	// executing the handler thereafter.
	Match(rctx *Context, method, path string) bool
}
