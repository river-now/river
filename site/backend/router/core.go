package router

import (
	"app"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/river-now/river/kit/middleware/etag"
	"github.com/river-now/river/kit/middleware/healthcheck"
	"github.com/river-now/river/kit/middleware/robotstxt"
	"github.com/river-now/river/kit/middleware/secureheaders"
	"github.com/river-now/river/kit/mux"
)

func Core() *mux.Router {
	r := mux.NewRouter(nil)

	mux.SetGlobalHTTPMiddleware(r, chimw.Logger)
	mux.SetGlobalHTTPMiddleware(r, chimw.Recoverer)
	mux.SetGlobalHTTPMiddleware(r, etag.Auto())
	mux.SetGlobalHTTPMiddleware(r, chimw.Compress(5))
	mux.SetGlobalHTTPMiddleware(r, app.Wave.ServeStatic(true))
	mux.SetGlobalHTTPMiddleware(r, secureheaders.Middleware)
	mux.SetGlobalHTTPMiddleware(r, healthcheck.Healthz)
	mux.SetGlobalHTTPMiddleware(r, robotstxt.Allow)
	mux.SetGlobalHTTPMiddleware(r, app.Wave.FaviconRedirect())

	// river UI routes
	mux.RegisterHandler(r, "GET", "/*", app.River.GetUIHandler(LoadersRouter))

	// river API routes
	actionsHandler := app.River.GetActionsHandler(ActionsRouter)
	mux.RegisterHandler(r, "GET", ActionsRouter.MountRoot("*"), actionsHandler)
	mux.RegisterHandler(r, "POST", ActionsRouter.MountRoot("*"), actionsHandler)

	return r
}
