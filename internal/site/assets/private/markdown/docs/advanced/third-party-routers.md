---
title: Using River With A Third-Party Router
description: Using River with a third-party router like Chi
---

The River framework's HTTP layer is built on top of its own `river/kit/mux`
package, and for most projects, we recommend using that for your core router.
It's fast, simple, and flexible.

However, if you prefer using a third-party router (such as
[Chi](https://go-chi.io/)), that is supported as well. All you need to do is add
`river.EnableThirdPartyRouter` in your middleware stack.

For example, here is how you might use River with Chi:

```go
package router

import (
	"app/control"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/river-now/river"
	"github.com/river-now/river/kit/middleware/etag"
	"github.com/river-now/river/kit/middleware/healthcheck"
	"github.com/river-now/river/kit/middleware/robotstxt"
	"github.com/river-now/river/kit/middleware/secureheaders"
)

var supportedAPIMethods = map[string]struct{}{
	"GET": {}, "POST": {}, "PUT": {}, "DELETE": {}, "PATCH": {},
}

func Core() *chi.Mux {
	r := chi.NewRouter()

	// Apply global middlewares
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(etag.Auto())
	r.Use(chimw.Compress(5))
	r.Use(control.Wave.ServeStatic(true))
	r.Use(secureheaders.Middleware)
	r.Use(healthcheck.Healthz)
	r.Use(robotstxt.Allow)
	r.Use(control.Wave.FaviconRedirect())

	// **IMPORTANT**: Add compat middleware
	r.Use(river.EnableThirdPartyRouter)

	// Register GET handler for loaders
	r.Method("GET", "/*", control.River.GetLoadersHandler(LoadersRouter))

	// Register handlers for API methods
	for method := range supportedAPIMethods {
		r.Method(
			method,
			ActionsRouter.MountRoot("*"),
			control.River.GetActionsHandler(ActionsRouter),
		)
	}

	return r
}
```
