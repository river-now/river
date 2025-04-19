package router

import (
	app "site"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/river-now/river/kit/middleware/healthcheck"
	"github.com/river-now/river/kit/middleware/robotstxt"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/tasks"
)

var sharedTasksRegistry = tasks.NewRegistry()

func Core() *mux.Router {
	r := mux.NewRouter(nil)

	mux.SetGlobalHTTPMiddleware(r, chimw.Logger)
	mux.SetGlobalHTTPMiddleware(r, chimw.Recoverer)
	mux.SetGlobalHTTPMiddleware(r, healthcheck.Healthz)
	mux.SetGlobalHTTPMiddleware(r, robotstxt.Allow)
	mux.SetGlobalHTTPMiddleware(r, app.Wave.FaviconRedirect())

	// static public assets
	mux.RegisterHandler(r, "GET", app.Wave.GetPublicPathPrefix()+"*", app.Wave.MustGetServeStaticHandler(true))

	// river UI routes
	mux.RegisterHandler(r, "GET", "/*", app.River.GetUIHandler(UIRouter))

	// river API routes
	actionsHandler := app.River.GetActionsHandler(ActionsRouter)
	mux.RegisterHandler(r, "GET", ActionsRouter.MountRoot("*"), actionsHandler)
	mux.RegisterHandler(r, "POST", ActionsRouter.MountRoot("*"), actionsHandler)

	return r
}
