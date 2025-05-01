package router

import (
	"net/http"
	app "site"
	"strings"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/river-now/river/kit/middleware/etag"
	"github.com/river-now/river/kit/middleware/healthcheck"
	"github.com/river-now/river/kit/middleware/robotstxt"
	"github.com/river-now/river/kit/middleware/secureheaders"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/tasks"
)

var sharedTasksRegistry = tasks.NewRegistry("site")

func Core() *mux.Router {
	r := mux.NewRouter(nil)

	mux.SetGlobalHTTPMiddleware(r, chimw.Logger)
	mux.SetGlobalHTTPMiddleware(r, chimw.Recoverer)
	mux.SetGlobalHTTPMiddleware(r, etag.Auto(&etag.Config{
		SkipFunc: func(r *http.Request) bool {
			return strings.HasPrefix(r.URL.Path, app.Wave.GetPublicPathPrefix())
		},
	}))
	mux.SetGlobalHTTPMiddleware(r, secureheaders.Middleware)
	mux.SetGlobalHTTPMiddleware(r, healthcheck.Healthz)
	mux.SetGlobalHTTPMiddleware(r, robotstxt.Allow)
	mux.SetGlobalHTTPMiddleware(r, app.Wave.FaviconRedirect())

	// static public assets
	mux.RegisterHandler(r, "GET", app.Wave.GetPublicPathPrefix()+"*", app.Wave.MustGetServeStaticHandler(true))

	// river UI routes
	mux.RegisterHandler(r, "GET", "/*", app.River.GetUIHandler(LoadersRouter))

	// river API routes
	actionsHandler := app.River.GetActionsHandler(ActionsRouter)
	mux.RegisterHandler(r, "GET", ActionsRouter.MountRoot("*"), actionsHandler)
	mux.RegisterHandler(r, "POST", ActionsRouter.MountRoot("*"), actionsHandler)

	return r
}
