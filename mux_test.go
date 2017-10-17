package fasthttpchi

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/valyala/fasthttp"
)

func TestMuxBasic(t *testing.T) {
	var count uint64

	countermw := func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			count++
			next(ctx)
		}
	}

	usermw := func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("user", "peter")
			next(ctx)
		}
	}

	exmw := func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("ex", "a")
			next(ctx)
		}
	}

	logbuf := bytes.Buffer{}
	logmsg := "logmw test"
	logmw := func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			logbuf.WriteString(logmsg)
			next(ctx)
		}
	}

	cxindex := func(ctx *fasthttp.RequestCtx) {
		user := ctx.UserValue("user")
		ctx.WriteString(fmt.Sprintf("hi %s", user))
	}

	ping := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(".")
	}

	headPing := func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("X-Ping", "1")
	}

	createPing := func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(201)
	}

	pingAll := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("ping all")
	}

	pingAll2 := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("ping all2")
	}

	pingOne := func(ctx *fasthttp.RequestCtx) {
		idParam := URLParam(ctx, "id")
		ctx.WriteString(fmt.Sprintf("ping one id: %s", idParam))
	}

	pingWoop := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("woop." + URLParam(ctx, "iidd"))
	}

	catchAll := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("catchall")
	}

	m := NewRouter()
	m.Use(countermw)
	m.Use(usermw)
	m.Use(exmw)
	m.Use(logmw)
	m.Get("/", cxindex)
	m.Method("GET", "/ping", ping)
	m.Method("GET", "/pingall", pingAll)
	m.Method("get", "/ping/all", pingAll)
	m.Get("/ping/all2", pingAll2)

	m.Head("/ping", headPing)
	m.Post("/ping", createPing)
	m.Get("/ping/{id}", pingWoop)
	m.Get("/ping/{id}", pingOne) // expected to overwrite to pingOne handler
	m.Get("/ping/{iidd}/woop", pingWoop)
	m.Handle("/admin/*", catchAll)

	e := newFastHTTPTester(t, m.ServeFastHTTP)

	// GET /
	e.GET("/").Expect().Status(200).Text().Equal("hi peter")
	if logbuf.String() != logmsg {
		t.Error("expecting log message from middleware:", logmsg)
	}

	// GET /ping
	e.GET("/ping").Expect().Status(200).Text().Equal(".")

	// GET /pingall
	e.GET("/pingall").Expect().Status(200).Text().Equal("ping all")

	// GET /ping/all
	e.GET("/ping/all").Expect().Status(200).Text().Equal("ping all")

	// GET /ping/all2
	e.GET("/ping/all2").Expect().Status(200).Text().Equal("ping all2")

	// GET /ping/123
	e.GET("/ping/123").Expect().Status(200).Text().Equal("ping one id: 123")

	// GET /ping/allan
	e.GET("/ping/allan").Expect().Status(200).Text().Equal("ping one id: allan")

	// GET /ping/1/woop
	e.GET("/ping/1/woop").Expect().Status(200).Text().Equal("woop.1")

	// HEAD /ping
	e.HEAD("/ping").Expect().Status(200).Header("X-Ping").Equal("1")

	// GET /admin/catch-this
	e.GET("/admin/catch-thazzzzz").Expect().Status(200).Text().Equal("catchall")

	// POST /admin/catch-this
	e.POST("/admin/catch-thazzzzz").Expect().Status(200).Text().Equal("catchall")

	// POST /ping/1/woop
	e.POST("/ping/1/woop").Expect().Status(405)

	// Custom Method
	e.Request("CUSTOM", "/not-exist").Expect().Status(405)
}

func TestMuxMounts(t *testing.T) {
	r := NewRouter()

	r.Get("/{hash}", func(ctx *fasthttp.RequestCtx) {
		v := URLParam(ctx, "hash")
		ctx.WriteString(fmt.Sprintf("/%s", v))
	})

	r.Route("/{hash}/share", func(r Router) {
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			v := URLParam(ctx, "hash")
			ctx.WriteString(fmt.Sprintf("/%s/share", v))
		})
		r.Get("/{network}", func(ctx *fasthttp.RequestCtx) {
			v := URLParam(ctx, "hash")
			n := URLParam(ctx, "network")
			ctx.WriteString(fmt.Sprintf("/%s/share/%s", v, n))
		})
	})

	m := NewRouter()
	m.Mount("/sharing", r)

	e := newFastHTTPTester(t, m.ServeFastHTTP)

	e.GET("/sharing/aBc").Expect().Status(200).Text().Equal("/aBc")
	e.GET("/sharing/aBc/share").Expect().Status(200).Text().Equal("/aBc/share")
	e.GET("/sharing/aBc/share/twitter").Expect().Status(200).Text().Equal("/aBc/share/twitter")
}

func TestMuxPlain(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("bye")
	})

	r.NotFound(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(404)
		ctx.WriteString("nothing here")
	})

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	e.GET("/hi").Expect().Status(200).Text().Equal("bye")
	e.GET("/nothing-here").Expect().Status(404).Text().Equal("nothing here")
}

func TestMuxEmptyRoutes(t *testing.T) {
	mux := NewRouter()

	apiRouter := NewRouter()
	// oops, we forgot to declare any route handlers

	mux.Mount("/api*", apiRouter)

	e := newFastHTTPTester(t, mux.ServeFastHTTP)
	e.GET("/").Expect().Status(404).Text().Equal("404 Page not found")

	func() {
		defer func() {
			if r := recover(); r != nil {
				if r != `fasthttp-chi: attempting to route to a mux with no handlers.` {
					t.Fatalf("expecting empty route panic")
				}
			}
		}()

		body := e.GET("/api").Expect().Text()
		t.Fatalf("oops, we are expecting a panic instead of getting resp: %s", body)
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				if r != `fasthttp-chi: attempting to route to a mux with no handlers.` {
					t.Fatalf("expecting empty route panic")
				}
			}
		}()

		body := e.GET("/api/abc").Expect().Text()
		t.Fatalf("oops, we are expecting a panic instead of getting resp: %s", body)
	}()
}

// Test a mux that routes a trailing slash, see also middleware/strip_test.go
// for an example of using a middleware to handle trailing slashes.
func TestMuxTrailingSlash(t *testing.T) {
	r := NewRouter()

	r.NotFound(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(404)
		ctx.WriteString("nothing here")
	})

	indexHandler := func(ctx *fasthttp.RequestCtx) {
		accountID := URLParam(ctx, "accountID")
		ctx.WriteString(accountID)
	}
	subRoutes := NewRouter()
	subRoutes.Get("/", indexHandler)

	r.Mount("/accounts/{accountID}", subRoutes)
	r.Get("/accounts/{accountID}/", indexHandler)

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	e.GET("/accounts/admin").Expect().Status(200).Text().Equal("admin")
	e.GET("/accounts/admin/").Expect().Status(200).Text().Equal("admin")
	e.GET("/nothing-here").Expect().Status(404).Text().Equal("nothing here")
}

func TestMuxNestedNotFound(t *testing.T) {
	r := NewRouter()

	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("mw", "mw")
			next(ctx)
		}
	})

	r.Get("/hi", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("bye")
	})

	r.With(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("with", "with")
			next(ctx)
		}
	}).NotFound(func(ctx *fasthttp.RequestCtx) {
		chkMw := ctx.UserValue("mw").(string)
		chkWith := ctx.UserValue("with").(string)
		ctx.SetStatusCode(404)
		ctx.WriteString(fmt.Sprintf("root 404 %s %s", chkMw, chkWith))
	})

	sr1 := NewRouter()
	sr1.Get("/sub", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("sub")
	})

	sr1.Group(func(r Router) {
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				ctx.SetUserValue("mw2", "mw2")
				next(ctx)
			}
		})

		r.NotFound(func(ctx *fasthttp.RequestCtx) {
			chkMw2 := ctx.UserValue("mw2").(string)
			ctx.SetStatusCode(404)
			ctx.WriteString(fmt.Sprintf("sub 404 %s", chkMw2))
		})
	})

	sr2 := NewRouter()
	sr2.Get("/sub", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("sub2")
	})

	r.Mount("/admin1", sr1)
	r.Mount("/admin2", sr2)

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	e.GET("/hi").Expect().Status(200).Text().Equal("bye")
	e.GET("/nothing-here").Expect().Status(404).Text().Equal("root 404 mw with")
	e.GET("/admin1/sub").Expect().Status(200).Text().Equal("sub")
	e.GET("/admin1/nop").Expect().Status(404).Text().Equal("sub 404 mw2")
	e.GET("/admin2/sub").Expect().Status(200).Text().Equal("sub2")

	// Not found pages should bubble up to the root.
	e.GET("/admin2/nope").Expect().Status(404).Text().Equal("root 404 mw with")
}

func TestMuxNestedMethodNotAllowed(t *testing.T) {
	r := NewRouter()

	r.Get("/root", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("root")
	})

	r.MethodNotAllowed(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(405)
		ctx.WriteString("root 405")
	})

	sr1 := NewRouter()
	sr1.Get("/sub1", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("sub1")
	})
	sr1.MethodNotAllowed(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(405)
		ctx.WriteString("sub1 405")
	})

	sr2 := NewRouter()
	sr2.Get("/sub2", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("sub2")
	})

	r.Mount("/prefix1", sr1)
	r.Mount("/prefix2", sr2)

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	e.GET("/root").Expect().Status(200).Text().Equal("root")
	e.PUT("/root").Expect().Status(405).Text().Equal("root 405")
	e.GET("/prefix1/sub1").Expect().Status(200).Text().Equal("sub1")
	e.PUT("/prefix1/sub1").Expect().Status(405).Text().Equal("sub1 405")
	e.GET("/prefix2/sub2").Expect().Status(200).Text().Equal("sub2")
	e.PUT("/prefix2/sub2").Expect().Status(405).Text().Equal("root 405")
}

func TestMuxComplicatedNotFound(t *testing.T) {
	// sub router with groups
	sub := NewRouter()
	sub.Route("/resource", func(r Router) {
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("private get")
		})
	})

	// Root router with groups
	r := NewRouter()
	r.Get("/auth", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("auth get")
	})
	r.Route("/public", func(r Router) {
		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("public get")
		})
	})
	r.Mount("/private", sub)
	r.NotFound(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(404)
		ctx.WriteString("custom not-found")
	})

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	// check that we didn't broke correct routes
	e.GET("/auth").Expect().Status(200).Text().Equal("auth get")
	e.GET("/public").Expect().Status(200).Text().Equal("public get")
	e.GET("/private/resource").Expect().Status(200).Text().Equal("private get")

	// check custom not-found on all levels
	e.GET("/nope").Expect().Status(404).Text().Equal("custom not-found")
	e.GET("/public/nope").Expect().Status(404).Text().Equal("custom not-found")
	e.GET("/private/nope").Expect().Status(404).Text().Equal("custom not-found")
	e.GET("/private/resource/nope").Expect().Status(404).Text().Equal("custom not-found")

	// check custom not-found on trailing slash routes
	e.GET("/auth/").Expect().Status(404).Text().Equal("custom not-found")
	e.GET("/public/").Expect().Status(404).Text().Equal("custom not-found")
	e.GET("/private/").Expect().Status(404).Text().Equal("custom not-found")
	e.GET("/private/resource/").Expect().Status(404).Text().Equal("custom not-found")
}

func TestMuxWith(t *testing.T) {
	var cmwInit1, cmwHandler1 uint64
	var cmwInit2, cmwHandler2 uint64

	mw1 := func(next HandlerFunc) HandlerFunc {
		cmwInit1++
		return func(ctx *fasthttp.RequestCtx) {
			cmwHandler1++
			ctx.SetUserValue("inline1", "yes")
			next(ctx)
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		cmwInit2++
		return func(ctx *fasthttp.RequestCtx) {
			cmwHandler2++
			ctx.SetUserValue("inline2", "yes")
			next(ctx)
		}
	}

	r := NewRouter()
	r.Get("/hi", func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("bye")
	})
	r.With(mw1).With(mw2).Get("/inline", func(ctx *fasthttp.RequestCtx) {
		v1 := ctx.UserValue("inline1").(string)
		v2 := ctx.UserValue("inline2").(string)
		ctx.WriteString(fmt.Sprintf("inline %s %s", v1, v2))
	})

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	e.GET("/hi").Expect().Status(200).Text().Equal("bye")
	e.GET("/inline").Expect().Status(200).Text().Equal("inline yes yes")
	if cmwInit1 != 1 {
		t.Fatalf("expecting cmwInit1 to be 1, got %d", cmwInit1)
	}
	if cmwHandler1 != 1 {
		t.Fatalf("expecting cmwHandler1 to be 1, got %d", cmwHandler1)
	}
	if cmwInit2 != 1 {
		t.Fatalf("expecting cmwInit2 to be 1, got %d", cmwInit2)
	}
	if cmwHandler2 != 1 {
		t.Fatalf("expecting cmwHandler2 to be 1, got %d", cmwHandler2)
	}
}

func TestMuxMiddlewareStack(t *testing.T) {
	var stdmwInit, stdmwHandler uint64
	stdmw := func(next HandlerFunc) HandlerFunc {
		stdmwInit++
		return func(ctx *fasthttp.RequestCtx) {
			stdmwHandler++
			next(ctx)
		}
	}

	var ctxmwInit, ctxmwHandler uint64
	ctxmw := func(next HandlerFunc) HandlerFunc {
		ctxmwInit++

		return func(ctx *fasthttp.RequestCtx) {
			ctxmwHandler++
			ctx.SetUserValue("count.ctxmwHandler", ctxmwHandler)
			next(ctx)
		}
	}

	var inCtxmwInit, inCtxmwHandler uint64
	inCtxmw := func(next HandlerFunc) HandlerFunc {
		inCtxmwInit++

		return func(ctx *fasthttp.RequestCtx) {
			inCtxmwHandler++
			next(ctx)
		}
	}

	r := NewRouter()
	r.Use(stdmw)
	r.Use(ctxmw)
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			if string(ctx.Path()) == "/ping" {
				ctx.WriteString("pong")
				return
			}
			next(ctx)
		}
	})

	var handlerCount uint64

	r.With(inCtxmw).Get("/", func(ctx *fasthttp.RequestCtx) {
		handlerCount++
		ctxmwHandlerCount := ctx.UserValue("count.ctxmwHandler").(uint64)
		str := fmt.Sprintf(
			"inits:%d reqs:%d ctxValue:%d",
			ctxmwInit,
			handlerCount,
			ctxmwHandlerCount,
		)
		ctx.WriteString(str)
	})

	e := newFastHTTPTester(t, r.ServeFastHTTP)
	e.GET("/").Expect()
	e.GET("/").Expect()
	e.GET("/").Expect().Status(200).Text().Equal("inits:1 reqs:3 ctxValue:3")
	e.GET("/ping").Expect().Status(200).Text().Equal("pong")
}

func TestMuxRouteGroups(t *testing.T) {
	var stdmwInit, stdmwHandler uint64

	stdmw := func(next HandlerFunc) HandlerFunc {
		stdmwInit++
		return func(ctx *fasthttp.RequestCtx) {
			stdmwHandler++
			next(ctx)
		}
	}

	var stdmwInit2, stdmwHandler2 uint64
	stdmw2 := func(next HandlerFunc) HandlerFunc {
		stdmwInit2++
		return func(ctx *fasthttp.RequestCtx) {
			stdmwHandler2++
			next(ctx)
		}
	}

	r := NewRouter()
	r.Group(func(r Router) {
		r.Use(stdmw)
		r.Get("/group", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("root group")
		})
	})
	r.Group(func(r Router) {
		r.Use(stdmw2)
		r.Get("/group2", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("root group2")
		})
	})

	e := newFastHTTPTester(t, r.ServeFastHTTP)

	// GET /group
	e.GET("/group").Expect().Status(200).Text().Equal("root group")
	if stdmwInit != 1 || stdmwHandler != 1 {
		t.Logf("stdmw counters failed, should be 1:1, got %d:%d", stdmwInit, stdmwHandler)
	}

	// GET /group2
	e.GET("/group2").Expect().Status(200).Text().Equal("root group2")
	if stdmwInit2 != 1 || stdmwHandler2 != 1 {
		t.Fatalf("stdmw2 counters failed, should be 1:1, got %d:%d", stdmwInit2, stdmwHandler2)
	}
}

func TestMuxBig(t *testing.T) {
	r := bigMux()

	e := newFastHTTPTester(t, r.ServeFastHTTP)
	e.GET("/favicon.ico").Expect().Status(200).Text().Equal("fav")
	e.GET("/hubs/4/view").Expect().Status(200).Text().Equal("/hubs/4/view reqid:1 session:anonymous")
	e.GET("/hubs/4/view/index.html").Expect().Status(200).Text().Equal("/hubs/4/view/index.html reqid:1 session:anonymous")
	e.GET("/hubs/ethereumhub/view/index.html").Expect().Status(200).Text().Equal("/hubs/ethereumhub/view/index.html reqid:1 session:anonymous")
	e.GET("/").Expect().Status(200).Text().Equal("/ reqid:1 session:elvis")
	e.GET("/suggestions").Expect().Status(200).Text().Equal("/suggestions reqid:1 session:elvis")
	e.GET("/woot/444/hiiii").Expect().Status(200).Text().Equal("/woot/444/hiiii")
	e.GET("/hubs/123").Expect().Status(200).Text().Equal("/hubs/123 reqid:1 session:elvis")
	e.GET("/hubs/123/touch").Expect().Status(200).Text().Equal("/hubs/123/touch reqid:1 session:elvis")
	e.GET("/hubs/123/webhooks").Expect().Status(200).Text().Equal("/hubs/123/webhooks reqid:1 session:elvis")
	e.GET("/hubs/123/posts").Expect().Status(200).Text().Equal("/hubs/123/posts reqid:1 session:elvis")
	e.GET("/folders").Expect().Status(404).Text().Equal("404 Page not found")
	e.GET("/folders/").Expect().Status(200).Text().Equal("/folders/ reqid:1 session:elvis")
	e.GET("/folders/public").Expect().Status(200).Text().Equal("/folders/public reqid:1 session:elvis")
	e.GET("/folders/nothing").Expect().Status(404).Text().Equal("404 Page not found")
}

func TestMuxSubroutesBasic(t *testing.T) {
	hIndex := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("index")
	}
	hArticlesList := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("articles-list")
	}
	hSearchArticles := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString("search-articles")
	}
	hGetArticle := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(fmt.Sprintf("get-article:%s", URLParam(ctx, "id")))
	}
	hSyncArticle := func(ctx *fasthttp.RequestCtx) {
		ctx.WriteString(fmt.Sprintf("sync-article:%s", URLParam(ctx, "id")))
	}

	r := NewRouter()
	r.Get("/", hIndex)
	r.Route("/articles", func(r Router) {
		r.Get("/", hArticlesList)
		r.Get("/search", hSearchArticles)
		r.Route("/{id}", func(r Router) {
			r.Get("/", hGetArticle)
			r.Get("/sync", hSyncArticle)
		})
	})

	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")
	// debugPrintTree(0, 0, r.tree, 0)
	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")

	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")
	// debugPrintTree(0, 0, rr1.tree, 0)
	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")

	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")
	// debugPrintTree(0, 0, rr2.tree, 0)
	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")

	e := newFastHTTPTester(t, r.ServeFastHTTP)
	e.GET("/").Expect().Status(200).Text().Equal("index")
	e.GET("/articles").Expect().Status(200).Text().Equal("articles-list")
	e.GET("/articles/search").Expect().Status(200).Text().Equal("search-articles")
	e.GET("/articles/123").Expect().Status(200).Text().Equal("get-article:123")
	e.GET("/articles/123/sync").Expect().Status(200).Text().Equal("sync-article:123")
}

/*
func TestMuxSubroutes(t *testing.T) {
	hHubView1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hub1"))
	})
	hHubView2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hub2"))
	})
	hHubView3 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hub3"))
	})
	hAccountView1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("account1"))
	})
	hAccountView2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("account2"))
	})

	r := NewRouter()
	r.Get("/hubs/{hubID}/view", hHubView1)
	r.Get("/hubs/{hubID}/view/*", hHubView2)

	sr := NewRouter()
	sr.Get("/", hHubView3)
	r.Mount("/hubs/{hubID}/users", sr)

	sr3 := NewRouter()
	sr3.Get("/", hAccountView1)
	sr3.Get("/hi", hAccountView2)

	var sr2 *Mux
	r.Route("/accounts/{accountID}", func(r Router) {
		sr2 = r.(*Mux)
		// r.Get("/", hAccountView1)
		r.Mount("/", sr3)
	})

	// This is the same as the r.Route() call mounted on sr2
	// sr2 := NewRouter()
	// sr2.Mount("/", sr3)
	// r.Mount("/accounts/{accountID}", sr2)

	ts := httptest.NewServer(r)
	defer ts.Close()

	var body, expected string
	var resp *http.Response

	_, body = testRequest(t, ts, "GET", "/hubs/123/view", nil)
	expected = "hub1"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/view/index.html", nil)
	expected = "hub2"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/users", nil)
	expected = "hub3"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	resp, body = testRequest(t, ts, "GET", "/hubs/123/users/", nil)
	expected = "404 page not found\n"
	if resp.StatusCode != 404 || body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/accounts/44", nil)
	expected = "account1"
	if body != expected {
		t.Fatalf("request:%s expected:%s got:%s", "GET /accounts/44", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/accounts/44/hi", nil)
	expected = "account2"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}

	// Test that we're building the routingPatterns properly
	router := r
	req, _ := http.NewRequest("GET", "/accounts/44/hi", nil)

	rctx := NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body = string(w.Body.Bytes())
	expected = "account2"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}

	routePatterns := rctx.RoutePatterns
	if len(rctx.RoutePatterns) != 3 {
		t.Fatalf("expected 3 routing patterns, got:%d", len(rctx.RoutePatterns))
	}
	expected = "/accounts/{accountID}/*"
	if routePatterns[0] != expected {
		t.Fatalf("routePattern, expected:%s got:%s", expected, routePatterns[0])
	}
	expected = "/*"
	if routePatterns[1] != expected {
		t.Fatalf("routePattern, expected:%s got:%s", expected, routePatterns[1])
	}
	expected = "/hi"
	if routePatterns[2] != expected {
		t.Fatalf("routePattern, expected:%s got:%s", expected, routePatterns[2])
	}

}

func TestSingleHandler(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := URLParam(r, "name")
		w.Write([]byte("hi " + name))
	})

	r, _ := http.NewRequest("GET", "/", nil)
	rctx := NewRouteContext()
	r = r.WithContext(context.WithValue(r.Context(), RouteCtxKey, rctx))
	rctx.URLParams.Add("name", "joe")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	body := string(w.Body.Bytes())
	expected := "hi joe"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
}

// TODO: a Router wrapper test..
//
// type ACLMux struct {
// 	*Mux
// 	XX string
// }
//
// func NewACLMux() *ACLMux {
// 	return &ACLMux{Mux: NewRouter(), XX: "hihi"}
// }
//
// // TODO: this should be supported...
// func TestWoot(t *testing.T) {
// 	var r Router = NewRouter()
//
// 	var r2 Router = NewACLMux() //NewRouter()
// 	r2.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte("hi"))
// 	})
//
// 	r.Mount("/", r2)
// }

func TestServeHTTPExistingContext(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
		s, _ := r.Context().Value(ctxKey{"testCtx"}).(string)
		w.Write([]byte(s))
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s, _ := r.Context().Value(ctxKey{"testCtx"}).(string)
		w.WriteHeader(404)
		w.Write([]byte(s))
	})

	testcases := []struct {
		Method         string
		Path           string
		Ctx            context.Context
		ExpectedStatus int
		ExpectedBody   string
	}{
		{
			Method:         "GET",
			Path:           "/hi",
			Ctx:            context.WithValue(context.Background(), ctxKey{"testCtx"}, "hi ctx"),
			ExpectedStatus: 200,
			ExpectedBody:   "hi ctx",
		},
		{
			Method:         "GET",
			Path:           "/hello",
			Ctx:            context.WithValue(context.Background(), ctxKey{"testCtx"}, "nothing here ctx"),
			ExpectedStatus: 404,
			ExpectedBody:   "nothing here ctx",
		},
	}

	for _, tc := range testcases {
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(tc.Method, tc.Path, nil)
		if err != nil {
			t.Fatalf("%v", err)
		}
		req = req.WithContext(tc.Ctx)
		r.ServeHTTP(resp, req)
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if resp.Code != tc.ExpectedStatus {
			t.Fatalf("%v != %v", tc.ExpectedStatus, resp.Code)
		}
		if string(b) != tc.ExpectedBody {
			t.Fatalf("%s != %s", tc.ExpectedBody, b)
		}
	}
}

func TestNestedGroups(t *testing.T) {
	handlerPrintCounter := func(w http.ResponseWriter, r *http.Request) {
		counter, _ := r.Context().Value(ctxKey{"counter"}).(int)
		w.Write([]byte(fmt.Sprintf("%v", counter)))
	}

	mwIncreaseCounter := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			counter, _ := ctx.Value(ctxKey{"counter"}).(int)
			counter++
			ctx = context.WithValue(ctx, ctxKey{"counter"}, counter)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Each route represents value of its counter (number of applied middlewares).
	r := NewRouter() // counter == 0
	r.Get("/0", handlerPrintCounter)
	r.Group(func(r Router) {
		r.Use(mwIncreaseCounter) // counter == 1
		r.Get("/1", handlerPrintCounter)

		// r.Handle(GET, "/2", Chain(mwIncreaseCounter).HandlerFunc(handlerPrintCounter))
		r.With(mwIncreaseCounter).Get("/2", handlerPrintCounter)

		r.Group(func(r Router) {
			r.Use(mwIncreaseCounter, mwIncreaseCounter) // counter == 3
			r.Get("/3", handlerPrintCounter)
		})
		r.Route("/", func(r Router) {
			r.Use(mwIncreaseCounter, mwIncreaseCounter) // counter == 3

			// r.Handle(GET, "/4", Chain(mwIncreaseCounter).HandlerFunc(handlerPrintCounter))
			r.With(mwIncreaseCounter).Get("/4", handlerPrintCounter)

			r.Group(func(r Router) {
				r.Use(mwIncreaseCounter, mwIncreaseCounter) // counter == 5
				r.Get("/5", handlerPrintCounter)
				// r.Handle(GET, "/6", Chain(mwIncreaseCounter).HandlerFunc(handlerPrintCounter))
				r.With(mwIncreaseCounter).Get("/6", handlerPrintCounter)

			})
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	for _, route := range []string{"0", "1", "2", "3", "4", "5", "6"} {
		if _, body := testRequest(t, ts, "GET", "/"+route, nil); body != route {
			t.Errorf("expected %v, got %v", route, body)
		}
	}
}

func TestMiddlewarePanicOnLateUse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello\n"))
	}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	defer func() {
		if recover() == nil {
			t.Error("expected panic()")
		}
	}()

	r := NewRouter()
	r.Get("/", handler)
	r.Use(mw) // Too late to apply middleware, we're expecting panic().
}

func TestMountingExistingPath(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}

	defer func() {
		if recover() == nil {
			t.Error("expected panic()")
		}
	}()

	r := NewRouter()
	r.Get("/", handler)
	r.Mount("/hi", http.HandlerFunc(handler))
	r.Mount("/hi", http.HandlerFunc(handler))
}

func TestMountingSimilarPattern(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bye"))
	})

	r2 := NewRouter()
	r2.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("foobar"))
	})

	r3 := NewRouter()
	r3.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("foo"))
	})

	r.Mount("/foobar", r2)
	r.Mount("/foo", r3)

	ts := httptest.NewServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/hi", nil); body != "bye" {
		t.Fatalf(body)
	}
}

func TestMuxMissingParams(t *testing.T) {
	r := NewRouter()
	r.Get(`/user/{userId:\d+}`, func(w http.ResponseWriter, r *http.Request) {
		userID := URLParam(r, "userId")
		w.Write([]byte(fmt.Sprintf("userId = '%s'", userID)))
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("nothing here"))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/user/123", nil); body != "userId = '123'" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/user/", nil); body != "nothing here" {
		t.Fatalf(body)
	}
}

func TestMuxContextIsThreadSafe(t *testing.T) {
	router := NewRouter()
	router.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Millisecond)
		defer cancel()

		<-ctx.Done()
	})

	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				w := httptest.NewRecorder()
				r, err := http.NewRequest("GET", "/ok", nil)
				if err != nil {
					t.Fatal(err)
				}

				ctx, cancel := context.WithCancel(r.Context())
				r = r.WithContext(ctx)

				go func() {
					cancel()
				}()
				router.ServeHTTP(w, r)
			}
		}()
	}
	wg.Wait()
}

func TestEscapedURLParams(t *testing.T) {
	m := NewRouter()
	m.Get("/api/{identifier}/{region}/{size}/{rotation}/*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		rctx := RouteContext(r.Context())
		if rctx == nil {
			t.Error("no context")
			return
		}
		identifier := URLParam(r, "identifier")
		if identifier != "http:%2f%2fexample.com%2fimage.png" {
			t.Errorf("identifier path parameter incorrect %s", identifier)
			return
		}
		region := URLParam(r, "region")
		if region != "full" {
			t.Errorf("region path parameter incorrect %s", region)
			return
		}
		size := URLParam(r, "size")
		if size != "max" {
			t.Errorf("size path parameter incorrect %s", size)
			return
		}
		rotation := URLParam(r, "rotation")
		if rotation != "0" {
			t.Errorf("rotation path parameter incorrect %s", rotation)
			return
		}
		w.Write([]byte("success"))
	})

	ts := httptest.NewServer(m)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/api/http:%2f%2fexample.com%2fimage.png/full/max/0/color.png", nil); body != "success" {
		t.Fatalf(body)
	}
}

func TestMuxMatch(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.Write([]byte("bye"))
	})
	r.Route("/articles", func(r Router) {
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := URLParam(r, "id")
			w.Header().Set("X-Article", id)
			w.Write([]byte("article:" + id))
		})
	})
	r.Route("/users", func(r Router) {
		r.Head("/{id}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-User", "-")
			w.Write([]byte("user"))
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := URLParam(r, "id")
			w.Header().Set("X-User", id)
			w.Write([]byte("user:" + id))
		})
	})

	tctx := NewRouteContext()

	tctx.Reset()
	if r.Match(tctx, "GET", "/users/1") == false {
		t.Fatal("expecting to find match for route:", "GET", "/users/1")
	}

	tctx.Reset()
	if r.Match(tctx, "HEAD", "/articles/10") == true {
		t.Fatal("not expecting to find match for route:", "HEAD", "/articles/10")
	}
}

func TestServerBaseContext(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		baseYes := r.Context().Value(ctxKey{"base"}).(string)
		if _, ok := r.Context().Value(http.ServerContextKey).(*http.Server); !ok {
			panic("missing server context")
		}
		if _, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); !ok {
			panic("missing local addr context")
		}
		w.Write([]byte(baseYes))
	})

	// Setup http Server with a base context
	ctx := context.WithValue(context.Background(), ctxKey{"base"}, "yes")
	ts := httptest.NewServer(ServerBaseContext(ctx, r))
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/", nil); body != "yes" {
		t.Fatalf(body)
	}
}
*/

/*----------  Internal  ----------*/

func bigMux() Router {
	r := NewRouter()

	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("requestID", "1")
			next(ctx)
		}
	})

	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(ctx *fasthttp.RequestCtx) {
			next(ctx)
		}
	})

	r.Group(func(r Router) {
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				ctx.SetUserValue("session.user", "anonymous")
				next(ctx)
			}
		})

		r.Get("/favicon.ico", func(ctx *fasthttp.RequestCtx) {
			ctx.WriteString("fav")
		})

		r.Get("/hubs/{hubID}/view", func(ctx *fasthttp.RequestCtx) {
			s := fmt.Sprintf(
				"/hubs/%s/view reqid:%s session:%s",
				URLParam(ctx, "hubID"),
				ctx.UserValue("requestID"),
				ctx.UserValue("session.user"),
			)
			ctx.WriteString(s)
		})

		r.Get("/hubs/{hubID}/view/*", func(ctx *fasthttp.RequestCtx) {
			s := fmt.Sprintf(
				"/hubs/%s/view/%s reqid:%s session:%s",
				URLParam(ctx, "hubID"),
				URLParam(ctx, "*"),
				ctx.UserValue("requestID"),
				ctx.UserValue("session.user"),
			)
			ctx.WriteString(s)
		})

		r.Post("/hubs/{hubSlug}/view/*", func(ctx *fasthttp.RequestCtx) {
			s := fmt.Sprintf(
				"/hubs/%s/view/%s reqid:%s session:%s",
				URLParam(ctx, "hubSlug"),
				URLParam(ctx, "*"),
				ctx.UserValue("requestID"),
				ctx.UserValue("session.user"),
			)
			ctx.WriteString(s)
		})
	})

	r.Group(func(r Router) {
		r.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx *fasthttp.RequestCtx) {
				ctx.SetUserValue("session.user", "elvis")
				next(ctx)
			}
		})

		r.Get("/", func(ctx *fasthttp.RequestCtx) {
			s := fmt.Sprintf(
				"/ reqid:%s session:%s",
				ctx.UserValue("requestID"),
				ctx.UserValue("session.user"),
			)
			ctx.WriteString(s)
		})

		r.Get("/suggestions", func(ctx *fasthttp.RequestCtx) {
			s := fmt.Sprintf(
				"/suggestions reqid:%s session:%s",
				ctx.UserValue("requestID"),
				ctx.UserValue("session.user"),
			)
			ctx.WriteString(s)
		})

		r.Get("/woot/{wootID}/*", func(ctx *fasthttp.RequestCtx) {
			s := fmt.Sprintf(
				"/woot/%s/%s",
				URLParam(ctx, "wootID"),
				URLParam(ctx, "*"),
			)
			ctx.WriteString(s)
		})

		r.Route("/hubs", func(r Router) {
			r.Route("/{hubID}", func(r Router) {
				r.Get("/", func(ctx *fasthttp.RequestCtx) {
					s := fmt.Sprintf(
						"/hubs/%s reqid:%s session:%s",
						URLParam(ctx, "hubID"),
						ctx.UserValue("requestID"),
						ctx.UserValue("session.user"),
					)
					ctx.WriteString(s)
				})

				r.Get("/touch", func(ctx *fasthttp.RequestCtx) {
					s := fmt.Sprintf(
						"/hubs/%s/touch reqid:%s session:%s",
						URLParam(ctx, "hubID"),
						ctx.UserValue("requestID"),
						ctx.UserValue("session.user"),
					)
					ctx.WriteString(s)
				})

				sr := NewRouter()

				sr.Get("/", func(ctx *fasthttp.RequestCtx) {
					s := fmt.Sprintf(
						"/hubs/%s/webhooks reqid:%s session:%s",
						URLParam(ctx, "hubID"),
						ctx.UserValue("requestID"),
						ctx.UserValue("session.user"),
					)
					ctx.WriteString(s)
				})

				sr.Route("/{webhookID}", func(r Router) {
					r.Get("/", func(ctx *fasthttp.RequestCtx) {
						s := fmt.Sprintf(
							"/hubs/%s/webhooks/%s reqid:%s session:%s",
							URLParam(ctx, "hubID"),
							URLParam(ctx, "webhookID"),
							ctx.UserValue("requestID"),
							ctx.UserValue("session.user"),
						)
						ctx.WriteString(s)
					})
				})

				r.Mount("/webhooks", Chain(func(next HandlerFunc) HandlerFunc {
					return func(ctx *fasthttp.RequestCtx) {
						ctx.SetUserValue("hook", true)
						next(ctx)
					}
				}).Handler(sr))

				r.Route("/posts", func(r Router) {
					r.Get("/", func(ctx *fasthttp.RequestCtx) {
						s := fmt.Sprintf(
							"/hubs/%s/posts reqid:%s session:%s",
							URLParam(ctx, "hubID"),
							ctx.UserValue("requestID"),
							ctx.UserValue("session.user"),
						)
						ctx.WriteString(s)
					})
				})
			})
		})

		r.Route("/folders/", func(r Router) {
			r.Get("/", func(ctx *fasthttp.RequestCtx) {
				s := fmt.Sprintf(
					"/folders/ reqid:%s session:%s",
					ctx.UserValue("requestID"),
					ctx.UserValue("session.user"),
				)
				ctx.WriteString(s)
			})

			r.Get("/public", func(ctx *fasthttp.RequestCtx) {
				s := fmt.Sprintf(
					"/folders/public reqid:%s session:%s",
					ctx.UserValue("requestID"),
					ctx.UserValue("session.user"),
				)
				ctx.WriteString(s)
			})
		})
	})

	return r
}

/*
func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}

func testHandler(t *testing.T, h http.Handler, method, path string, body io.Reader) (*http.Response, string) {
	r, _ := http.NewRequest(method, path, body)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w.Result(), string(w.Body.Bytes())
}

type testFileSystem struct {
	open func(name string) (http.File, error)
}

func (fs *testFileSystem) Open(name string) (http.File, error) {
	return fs.open(name)
}

type testFile struct {
	name     string
	contents []byte
}

func (tf *testFile) Close() error {
	return nil
}

func (tf *testFile) Read(p []byte) (n int, err error) {
	copy(p, tf.contents)
	return len(p), nil
}

func (tf *testFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (tf *testFile) Readdir(count int) ([]os.FileInfo, error) {
	stat, _ := tf.Stat()
	return []os.FileInfo{stat}, nil
}

func (tf *testFile) Stat() (os.FileInfo, error) {
	return &testFileInfo{tf.name, int64(len(tf.contents))}, nil
}

type testFileInfo struct {
	name string
	size int64
}

func (tfi *testFileInfo) Name() string       { return tfi.name }
func (tfi *testFileInfo) Size() int64        { return tfi.size }
func (tfi *testFileInfo) Mode() os.FileMode  { return 0755 }
func (tfi *testFileInfo) ModTime() time.Time { return time.Now() }
func (tfi *testFileInfo) IsDir() bool        { return false }
func (tfi *testFileInfo) Sys() interface{}   { return nil }

type ctxKey struct {
	name string
}

func (k ctxKey) String() string {
	return "context value " + k.name
}

func BenchmarkMux(b *testing.B) {
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h3 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h4 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h5 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h6 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	mx := NewRouter()
	mx.Get("/", h1)
	mx.Get("/hi", h2)
	mx.Get("/sup/{id}/and/{this}", h3)

	mx.Route("/sharing/{hash}", func(mx Router) {
		mx.Get("/", h4)          // subrouter-1
		mx.Get("/{network}", h5) // subrouter-1
		mx.Get("/twitter", h5)
		mx.Route("/direct", func(mx Router) {
			mx.Get("/", h6) // subrouter-2
		})
	})

	routes := []string{
		"/",
		"/sup/123/and/this",
		"/sharing/aBc",         // subrouter-1
		"/sharing/aBc/twitter", // subrouter-1
		"/sharing/aBc/direct",  // subrouter-2
	}

	for _, path := range routes {
		b.Run("route:"+path, func(b *testing.B) {
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", path, nil)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				mx.ServeHTTP(w, r)
			}
		})
	}
}
*/

func newFastHTTPTester(t *testing.T, h HandlerFunc) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		// Pass requests directly to FastHTTPHandler.
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(fasthttp.RequestHandler(h)),
			Jar:       httpexpect.NewJar(),
		},
		// Report errors using testify.
		Reporter: httpexpect.NewAssertReporter(t),
	})
}
