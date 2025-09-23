package river

import (
	_ "embed"

	rf "github.com/river-now/river/internal/framework"
	"github.com/river-now/river/kit/headels"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/parseutil"
	"github.com/river-now/river/wave"
)

/////////////////////////////////////////////////////////////////////
/////// PUBLIC API
/////////////////////////////////////////////////////////////////////

type (
	River                             = rf.River
	HeadEls                           = headels.HeadEls
	AdHocType                         = rf.AdHocType
	RiverAppConfig                    = rf.RiverAppConfig
	LoadersRouter                     = rf.LoadersRouter
	LoaderReqData                     = rf.LoaderReqData
	ActionsRouter                     = rf.ActionsRouter
	ActionReqData[I any]              = rf.ActionReqData[I]
	None                              = mux.None
	Action[I any, O any]              = rf.TaskHandler[I, O]
	Loader[O any]                     = rf.TaskHandler[None, O]
	BuildRiverOptions                 = rf.BuildRiverOptions
	LoaderFunc[Ctx any, O any]        = func(*Ctx) (O, error)
	ActionFunc[Ctx any, I any, O any] = func(*Ctx) (O, error)
	LoadersRouterOptions              = rf.LoadersRouterOptions
	ActionsRouterOptions              = rf.ActionsRouterOptions
)

var (
	// Wave convenience re-exports
	MustGetPort  = wave.MustGetPort
	GetIsDev     = wave.GetIsDev
	SetModeToDev = wave.SetModeToDev

	IsJSONRequest          = rf.IsJSONRequest
	NewHeadEls             = headels.New
	RiverBuildIDHeaderKey  = rf.RiverBuildIDHeaderKey
	EnableThirdPartyRouter = mux.InjectTasksCtxMiddleware
	NewLoadersRouter       = rf.NewLoadersRouter
	NewActionsRouter       = rf.NewActionsRouter
)

func NewRiverApp(o RiverAppConfig) *River { return rf.NewRiverApp(o) }

func NewLoader[O any, CtxPtr ~*Ctx, Ctx any](
	r *LoadersRouter,
	p string,
	f func(CtxPtr) (O, error),
	ctxFactory func(*LoaderReqData) CtxPtr,
) *Loader[O] {
	wrappedF := func(c *LoaderReqData) (O, error) { return f(ctxFactory(c)) }
	loaderTask := mux.TaskHandlerFromFunc(wrappedF)
	mux.RegisterNestedTaskHandler(r.NestedRouter, p, loaderTask)
	return loaderTask
}

func NewAction[I any, O any, CtxPtr ~*Ctx, Ctx any](
	r *ActionsRouter,
	m string,
	p string,
	f func(CtxPtr) (O, error),
	ctxFactory func(*mux.ReqData[I]) CtxPtr,
) *Action[I, O] {
	wrappedF := func(c *mux.ReqData[I]) (O, error) { return f(ctxFactory(c)) }
	actionTask := mux.TaskHandlerFromFunc(wrappedF)
	mux.RegisterTaskHandler(r.Router, m, p, actionTask)
	return actionTask
}

//go:embed package.json
var packageJSON string

// This utility exists primarily in service of the River.now
// website. There is no guarantee that this utility will always
// be available or kept up to date.
func Internal__GetCurrentNPMVersion() string {
	_, _, currentVersion := parseutil.PackageJSONFromString(packageJSON)
	return currentVersion
}
