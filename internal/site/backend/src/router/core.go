package router

import (
	"net/http"
	"site/backend"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/river-now/river/kit/middleware/etag"
	"github.com/river-now/river/kit/middleware/healthcheck"
	"github.com/river-now/river/kit/middleware/robotstxt"
	"github.com/river-now/river/kit/middleware/secureheaders"
	"github.com/river-now/river/kit/mux"
)

func Init() (http.Handler, string) {
	r := mux.NewRouter()
	app := backend.App.Init()

	mux.SetGlobalHTTPMiddleware(r, chimw.Logger)
	mux.SetGlobalHTTPMiddleware(r, chimw.Recoverer)
	mux.SetGlobalHTTPMiddleware(r, etag.Auto())
	mux.SetGlobalHTTPMiddleware(r, chimw.Compress(5))
	mux.SetGlobalHTTPMiddleware(r, app.ServeStatic())
	mux.SetGlobalHTTPMiddleware(r, secureheaders.Middleware)
	mux.SetGlobalHTTPMiddleware(r, healthcheck.Healthz)
	mux.SetGlobalHTTPMiddleware(r, robotstxt.Allow)

	loaders := app.Loaders(LoadersRouter)
	mux.RegisterHandler(r, "GET", loaders.HandlerMountPattern(), loaders.Handler())

	actions := app.Actions(ActionsRouter)
	for m := range actions.SupportedMethods() {
		mux.RegisterHandler(r, m, actions.HandlerMountPattern(), actions.Handler())
	}

	return r, app.ServerAddr()
}
