package router

import (
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/river-now/river/kit/middleware/etag"
	"github.com/river-now/river/kit/middleware/healthcheck"
	"github.com/river-now/river/kit/middleware/robotstxt"
	"github.com/river-now/river/kit/middleware/secureheaders"
	"github.com/river-now/river/kit/mux"
)

func Init() (addr string, handler http.Handler) {
	App.Init()

	r := mux.NewRouter()
	loaders, actions := App.Loaders(), App.Actions()

	mux.SetGlobalHTTPMiddleware(r, chimw.Logger)
	mux.SetGlobalHTTPMiddleware(r, chimw.Recoverer)
	mux.SetGlobalHTTPMiddleware(r, etag.Auto())
	mux.SetGlobalHTTPMiddleware(r, chimw.Compress(5))
	mux.SetGlobalHTTPMiddleware(r, App.ServeStatic())
	mux.SetGlobalHTTPMiddleware(r, secureheaders.Middleware)
	mux.SetGlobalHTTPMiddleware(r, healthcheck.Healthz)
	mux.SetGlobalHTTPMiddleware(r, robotstxt.Allow)

	mux.RegisterHandler(r, "GET", loaders.HandlerMountPattern(), loaders.Handler())

	for m := range actions.SupportedMethods() {
		mux.RegisterHandler(r, m, actions.HandlerMountPattern(), actions.Handler())
	}

	return App.ServerAddr(), r
}
