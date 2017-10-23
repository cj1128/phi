package phi

import (
	"net/http"
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/valyala/fasthttp"
)

func TestMuxBasic(t *testing.T) {
	r := NewRouter()
	h := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("ok")
	}
	r.Connect("/connect", h)
	r.Delete("/delete", h)
	r.Get("/get", h)
	r.Head("/head", h)
	r.Options("/options", h)
	r.Patch("/patch", h)
	r.Post("/post", h)
	r.Put("/put", h)
	r.Trace("/trace", h)

	r.Method("GET", "/method-get", h)
	r.Handle("/handle", h)

	e := newFastHTTPTester(t, r)
	e.Request("CONNECT", "/connect").Expect().Status(200).Text().Equal("ok")
	e.DELETE("/delete").Expect().Status(200).Text().Equal("ok")
	e.GET("/get").Expect().Status(200).Text().Equal("ok")
	e.HEAD("/head").Expect().Status(200).Text().Equal("ok")
	e.OPTIONS("/options").Expect().Status(200).Text().Equal("ok")
	e.PATCH("/patch").Expect().Status(200).Text().Equal("ok")
	e.POST("/post").Expect().Status(200).Text().Equal("ok")
	e.PUT("/put").Expect().Status(200).Text().Equal("ok")
	e.PUT("/put").Expect().Status(200).Text().Equal("ok")
	e.Request("TRACE", "/trace").Expect().Status(200).Text().Equal("ok")

	e.GET("/method-get").Expect().Status(200).Text().Equal("ok")

	e.Request("CONNECT", "/handle").Expect().Status(200).Text().Equal("ok")
	e.DELETE("/handle").Expect().Status(200).Text().Equal("ok")
	e.GET("/handle").Expect().Status(200).Text().Equal("ok")
	e.HEAD("/handle").Expect().Status(200).Text().Equal("ok")
	e.OPTIONS("/handle").Expect().Status(200).Text().Equal("ok")
	e.PATCH("/handle").Expect().Status(200).Text().Equal("ok")
	e.POST("/handle").Expect().Status(200).Text().Equal("ok")
	e.PUT("/handle").Expect().Status(200).Text().Equal("ok")
	e.PUT("/handle").Expect().Status(200).Text().Equal("ok")
	e.Request("TRACE", "/handle").Expect().Status(200).Text().Equal("ok")
}

func TestMuxURLParams(t *testing.T) {
	r := NewRouter()

	r.Get("/{name}", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(URLParam(ctx, "name"))
	})
	r.Get("/sub/{name}", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("sub" + URLParam(ctx, "name"))
	})

	e := newFastHTTPTester(t, r)
	e.GET("/hello").Expect().Status(200).Text().Equal("hello")
	e.GET("/hello/all").Expect().Status(404)
	e.GET("/sub/hello").Expect().Status(200).Text().Equal("subhello")
}

func TestMuxUse(t *testing.T) {
	r := NewRouter()
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
			ctx.WriteString("+mw1")
		}
	})
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
			ctx.WriteString("+mw2")
		}
	})
	r.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("ok")
	})

	e := newFastHTTPTester(t, r)
	e.GET("/").Expect().Status(200).Text().Equal("ok+mw2+mw1")
	e.GET("/nothing").Expect().Status(404).Text().Equal("404 Page not found+mw2+mw1")
}

func TestMuxWith(t *testing.T) {
	r := NewRouter()
	h := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("ok")
	}
	r.Get("/", h)
	mw := func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
			ctx.WriteString("+with")
		}
	}
	r.With(mw).Get("/with", h)

	e := newFastHTTPTester(t, r)
	e.GET("/").Expect().Status(200).Text().Equal("ok")
	e.GET("/with").Expect().Status(200).Text().Equal("ok+with")
}

func TestMuxGroup(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("index")
	})
	r.Group(func(r Router) {
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				next(ctx)
				ctx.WriteString("+group")
			}
		})
		r.Get("/s1", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("s1")
		})
		r.Get("/s2", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("s2")
		})
	})

	e := newFastHTTPTester(t, r)
	e.GET("/").Expect().Status(200).Text().Equal("index")
	e.GET("/s1").Expect().Status(200).Text().Equal("s1+group")
	e.GET("/s2").Expect().Status(200).Text().Equal("s2+group")
}

func TestMuxRoute(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("index")
	})
	r.Route("/admin", func(r Router) {
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				next(ctx)
				ctx.WriteString("+route")
			}
		})
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("admin")
		})
		r.Get("/s1", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("s1")
		})
	})
	e := newFastHTTPTester(t, r)
	e.GET("/").Expect().Status(200).Text().Equal("index")
	e.GET("/admin").Expect().Status(200).Text().Equal("admin+route")
	e.GET("/admin/s1").Expect().Status(200).Text().Equal("s1+route")
}

func TestMuxMount(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("index")
	})

	sub := NewRouter()
	sub.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
			ctx.WriteString("+mount")
		}
	})
	sub.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("admin")
	})
	sub.Get("/s1", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("s1")
	})

	r.Mount("/admin", sub)
	e := newFastHTTPTester(t, r)
	e.GET("/").Expect().Status(200).Text().Equal("index")
	e.GET("/admin").Expect().Status(200).Text().Equal("admin+mount")
	e.GET("/admin/s1").Expect().Status(200).Text().Equal("s1+mount")
}

func TestMuxNotFound(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		r := NewRouter()
		r.NotFound(func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(404)
			ctx.WriteString("not found")
		})
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("ok")
		})
		e := newFastHTTPTester(t, r)
		e.GET("/").Expect().Status(200).Text().Equal("ok")
		e.GET("/no").Expect().Status(404).Text().Equal("not found")
		e.GET("/nono").Expect().Status(404).Text().Equal("not found")
	})

	t.Run("nested", func(t *testing.T) {
		r := NewRouter()
		r.NotFound(func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("not found")
			ctx.SetStatusCode(404)
		})

		h := func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("ok")
		}

		// should copy parent NotFound if none
		r.Route("/s1", func(r Router) {
			r.Get("/", h)
		})

		sub := NewRouter()
		sub.NotFound(func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("sub not found")
			ctx.SetStatusCode(404)
		})
		sub.Get("/", h)
		r.Mount("/s2", sub)

		e := newFastHTTPTester(t, r)
		e.GET("/no").Expect().Status(404).Text().Equal("not found")
		e.GET("/s1/no").Expect().Status(404).Text().Equal("not found")
		e.GET("/s2/no").Expect().Status(404).Text().Equal("sub not found")
	})
}

func TestMuxMethodNotAllowed(t *testing.T) {
	t.Run("simple case", func(t *testing.T) {
		r := NewRouter()
		r.MethodNotAllowed(func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("bad method")
			ctx.SetStatusCode(405)
		})
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("ok")
		})

		e := newFastHTTPTester(t, r)
		e.GET("/").Expect().Status(200).Text().Equal("ok")
		e.POST("/").Expect().Status(405).Text().Equal("bad method")
	})

	t.Run("nested", func(t *testing.T) {
		r := NewRouter()
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("index")
		})
		r.MethodNotAllowed(func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("bad method")
			ctx.SetStatusCode(405)
		})

		// should copy parent MethodNotAllowed if none
		r.Route("/s1", func(r Router) {
			r.Get("/", func(ctx *fasthttp.RequestCtx) {
				ctx.WriteString("s1")
			})
		})

		sub := NewRouter()
		sub.MethodNotAllowed(func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("s2 bad method")
			ctx.SetStatusCode(405)
		})
		sub.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("s2")
		})
		r.Mount("/s2", sub)

		e := newFastHTTPTester(t, r)
		e.POST("/").Expect().Status(405).Text().Equal("bad method")
		e.POST("/s1").Expect().Status(405).Text().Equal("bad method")
		e.POST("/s2").Expect().Status(405).Text().Equal("s2 bad method")
	})
}

func TestMuxBigMux(t *testing.T) {
	r := bigMux()
	e := newFastHTTPTester(t, r)

	e.GET("/").Expect().Status(200).Text().Equal("index+reqid=1")
	e.POST("/").Expect().Status(405).Text().Equal("whoops, bad method+reqid=1")
	e.GET("/nothing").Expect().Status(404).Text().Equal("whoops, not found+reqid=1")

	// task
	e.GET("/task").Expect().Status(200).Text().Equal("task+task+reqid=1")
	e.POST("/task").Expect().Status(200).Text().Equal("new task+task+reqid=1")
	e.DELETE("/task").Expect().Status(200).Text().Equal("delete task+caution+task+reqid=1")

	// cat
	e.GET("/cat").Expect().Status(200).Text().Equal("cat+cat+reqid=1")
	e.PATCH("/cat").Expect().Status(200).Text().Equal("patch cat+cat+reqid=1")
	e.GET("/cat/nothing").Expect().Status(404).Text().Equal("no such cat+cat+reqid=1")

	// user
	e.GET("/user").Expect().Status(200).Text().Equal("user+user+reqid=1")
	e.POST("/user").Expect().Status(200).Text().Equal("new user+user+reqid=1")
	e.GET("/user/nothing").Expect().Status(404).Text().Equal("no such user+user+reqid=1")
}

/*----------  Internal  ----------*/

func bigMux() Router {
	r := NewRouter()

	reqIDMW := func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
			ctx.WriteString("+reqid=1")
		}
	}
	r.Use(reqIDMW)

	r.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("index")
	})
	r.NotFound(func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("whoops, not found")
		ctx.SetStatusCode(404)
	})
	r.MethodNotAllowed(func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("whoops, bad method")
		ctx.SetStatusCode(405)
	})

	// tasks
	r.Group(func(r Router) {
		mw := func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				next(ctx)
				ctx.WriteString("+task")
			}
		}
		r.Use(mw)

		r.Get("/task", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("task")
		})
		r.Post("/task", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("new task")
		})

		caution := func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				next(ctx)
				ctx.WriteString("+caution")
			}
		}
		r.With(caution).Delete("/task", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("delete task")
		})
	})

	// cat
	r.Route("/cat", func(r Router) {
		r.NotFound(func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("no such cat")
			ctx.SetStatusCode(404)
		})
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				next(ctx)
				ctx.WriteString("+cat")
			}
		})
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("cat")
		})
		r.Patch("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("patch cat")
		})
	})

	// user
	userRouter := NewRouter()
	userRouter.NotFound(func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("no such user")
		ctx.SetStatusCode(404)
	})
	userRouter.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
			ctx.WriteString("+user")
		}
	})
	userRouter.Get("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("user")
	})
	userRouter.Post("/", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("new user")
	})
	r.Mount("/user", userRouter)

	return r
}

func newFastHTTPTester(t *testing.T, h Handler) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		// Pass requests directly to FastHTTPHandler.
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(fasthttp.RequestHandler(h.ServeFastHTTP)),
			Jar:       httpexpect.NewJar(),
		},
		// Report errors using testify.
		Reporter: httpexpect.NewAssertReporter(t),
	})
}
