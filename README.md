# fasthttp chi

port [chi](https://github.com/go-chi/chi) to fasthttp.

## Example

```go
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
```

output:

|        Path         | Status Code |               Body               |
| :-----------------: | :---------: | :------------------------------: |
|       `GET /`       |     200     |          index+reqid=1           |
|      `POST /`       |     405     |    whoops, bad method+reqid=1    |
|   `GET /nothing`    |     404     |    whoops, not found+reqid=1     |
|     `GET /task`     |     200     |        task+task+reqid=1         |
|    `POST /task`     |     200     |      new task+task+reqid=1       |
|   `DELETE /task`    |     200     | delete task+caution+task+reqid=1 |
|     `GET /cat`      |     200     |         cat+cat+reqid=1          |
|    `PATCH /cat`     |     200     |      patch cat+cat+reqid=1       |
| `GET /cat/nothing`  |     404     |     no such cat+cat+reqid=1      |
|     `GET /user`     |     200     |        user+user+reqid=1         |
|    `POST /user`     |     200     |      new user+user+reqid=1       |
| `GET /user/nothing` |     404     |    no such user+user+reqid=1     |

