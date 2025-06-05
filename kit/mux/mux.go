package mux

import (
	"net/http"
	"path"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/river-now/river/kit/colorlog"
	"github.com/river-now/river/kit/contextutil"
	"github.com/river-now/river/kit/genericsutil"
	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/opt"
	"github.com/river-now/river/kit/reflectutil"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/tasks"
	"github.com/river-now/river/kit/validate"
)

// __TODO test injectTasksCtx and TasksCtxRequirer

var (
	muxLog           = colorlog.New("mux")
	requestStore     = contextutil.NewStore[*rdTransport]("__river_kit_mux_request_data")
	emptyParams      = make(Params, 0)
	emptyHTTPMws     = []httpMiddlewareWithOptions{}
	emptyTaskMws     = []taskMiddlewareWithOptions{}
	emptySplatValues = []string{}
)

type (
	None                      = genericsutil.None
	TaskHandler[I any, O any] = tasks.Task[*ReqData[I], O]
	Params                    = matcher.Params
)

type rdTransport struct {
	params        Params
	splatVals     []string
	tasksCtx      *tasks.TasksCtx
	req           *http.Request
	responseProxy *response.Proxy
}

type ReqData[I any] struct {
	params        Params
	splatVals     []string
	tasksCtx      *tasks.TasksCtx
	input         I
	req           *http.Request
	responseProxy *response.Proxy
}

/*
Order of registration of handlers does not matter. Order of middleware
registration DOES matter. For traditional middleware, it will run sequentially,
first to last. For task middleware, they will run with maximum parallelism, but
their response proxies will be merged according to the rules of response.Proxy.
*/

type MiddlewareOptions struct {
	// Return true if the middleware should be run for this request.
	// If nil, the middleware will always run.
	If func(r *http.Request) bool
}

type (
	HTTPMiddleware                = func(http.Handler) http.Handler
	TaskMiddlewareFunc[O any]     = genericsutil.IOFunc[*ReqData[None], O]
	TaskMiddleware[O any]         = tasks.Task[*ReqData[None], O]
	TaskHandlerFunc[I any, O any] = genericsutil.IOFunc[*ReqData[I], O]
)

type httpMiddlewareWithOptions struct {
	mw   HTTPMiddleware
	opts *MiddlewareOptions
}

type taskMiddlewareWithOptions struct {
	mw   tasks.AnyTask
	opts *MiddlewareOptions
}

type Router struct {
	marshalInput       func(r *http.Request, iPtr any) error
	httpMws            []httpMiddlewareWithOptions
	taskMws            []taskMiddlewareWithOptions
	methodToMatcherMap map[string]*methodMatcher
	matcherOpts        *matcher.Options
	notFoundHandler    http.Handler
	mountRoot          string
	allRoutes          []AnyRoute
	injectTasksCtx     bool
}

func (rt *Router) AllRoutes() []AnyRoute { return rt.allRoutes }

func (rt *Router) GetExplicitIndexSegment() string { return rt.matcherOpts.ExplicitIndexSegment }
func (rt *Router) GetDynamicParamPrefixRune() rune { return rt.matcherOpts.DynamicParamPrefixRune }
func (rt *Router) GetSplatSegmentRune() rune       { return rt.matcherOpts.SplatSegmentRune }

// Takes zero or one pattern strings. If no arguments are provided, returns
// the mount root, otherwise returns the mount root joined with the
// provided pattern. Discards any extra arguments. For example, if
// mux.MountRoot() were to return "/api/", then mux.MountRoot("foo") would
// return "/api/foo", and mux.MountRoot("foo", "bar") would still just
// return "/api/foo".
func (rt *Router) MountRoot(optionalPatternToAppend ...string) string {
	if len(optionalPatternToAppend) == 0 {
		return rt.mountRoot
	}
	return path.Join(rt.mountRoot, optionalPatternToAppend[0])
}

type methodMatcher struct {
	matcher        *matcher.Matcher
	httpMws        []httpMiddlewareWithOptions
	taskMws        []taskMiddlewareWithOptions
	routes         map[string]AnyRoute
	reqDataGetters map[string]reqDataGetter
}

type TasksCtxRequirer interface {
	http.Handler
	NeedsTasksCtx()
}

var HandlerNeedsTasksCtxImplReflectType = reflectutil.ToInterfaceReflectType[TasksCtxRequirer]()

type TasksCtxRequirerFunc func(http.ResponseWriter, *http.Request)

func (h TasksCtxRequirerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) { h(w, r) }
func (h TasksCtxRequirerFunc) NeedsTasksCtx()                                   {}

type Options struct {
	// Used for mounting a router at a specific path, e.g., "/api/". If set,
	// the router will strip the provided mount root from the beginning of
	// incoming url paths before matching them against registered patterns.
	MountRoot string

	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.

	// Required if using task handlers. Do validation or whatever you want here,
	// and mutate the input ptr to the desired value (this is what will ultimately
	// be returned by c.Input()).
	MarshalInput func(r *http.Request, inputPtr any) error

	// If true, automatically injects a TasksCtx into the request context.
	InjectTasksCtx bool
}

func NewRouter(opts *Options) *Router {
	matcherOpts := new(matcher.Options)

	if opts == nil {
		opts = new(Options)
	}

	matcherOpts.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	matcherOpts.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')

	mountRootToUse := opts.MountRoot

	if mountRootToUse != "" {
		if len(mountRootToUse) == 1 && mountRootToUse[0] == '/' {
			mountRootToUse = ""
		}
		if len(mountRootToUse) > 1 && mountRootToUse[0] != '/' {
			mountRootToUse = "/" + mountRootToUse
		}
		if len(mountRootToUse) > 0 && mountRootToUse[len(mountRootToUse)-1] != '/' {
			mountRootToUse = mountRootToUse + "/"
		}
	}

	return &Router{
		marshalInput:       opts.MarshalInput,
		methodToMatcherMap: make(map[string]*methodMatcher),
		matcherOpts:        matcherOpts,
		mountRoot:          mountRootToUse,
		httpMws:            emptyHTTPMws,
		taskMws:            emptyTaskMws,
		injectTasksCtx:     opts.InjectTasksCtx,
	}
}

// TaskHandlers are used for JSON responses only, and they are intended to
// be particularly convenient for sending JSON. If you need to send a different
// content type, use a traditional http.Handler instead.
func TaskHandlerFromFunc[I any, O any](taskHandlerFunc TaskHandlerFunc[I, O]) *TaskHandler[I, O] {
	return tasks.NewTask(func(c *tasks.TasksCtx, rd *ReqData[I]) (O, error) {
		return taskHandlerFunc(rd)
	})
}

func TaskMiddlewareFromFunc[O any](userFunc TaskMiddlewareFunc[O]) *TaskMiddleware[O] {
	return tasks.NewTask(func(c *tasks.TasksCtx, rd *ReqData[None]) (O, error) {
		return userFunc(rd)
	})
}

func SetGlobalTaskMiddleware[O any](router *Router, taskMw *TaskMiddleware[O], opts ...*MiddlewareOptions) {
	router.taskMws = append(router.taskMws, taskMiddlewareWithOptions{
		mw:   taskMw,
		opts: getFirstOpt(opts),
	})
}

func SetGlobalHTTPMiddleware(router *Router, httpMw HTTPMiddleware, opts ...*MiddlewareOptions) {
	router.httpMws = append(router.httpMws, httpMiddlewareWithOptions{
		mw:   httpMw,
		opts: getFirstOpt(opts),
	})
}

func SetMethodLevelTaskMiddleware[O any](
	router *Router, method string, taskMw *TaskMiddleware[O], opts ...*MiddlewareOptions,
) {
	mm := router.getOrCreateMethodMatcher(method)
	mm.taskMws = append(mm.taskMws, taskMiddlewareWithOptions{
		mw:   taskMw,
		opts: getFirstOpt(opts),
	})
}

func SetMethodLevelHTTPMiddleware(router *Router, method string, httpMw HTTPMiddleware, opts ...*MiddlewareOptions) {
	mm := router.getOrCreateMethodMatcher(method)
	mm.httpMws = append(mm.httpMws, httpMiddlewareWithOptions{
		mw:   httpMw,
		opts: getFirstOpt(opts),
	})
}

func SetPatternLevelTaskMiddleware[PI any, PO any, MWO any](route *Route[PI, PO], taskMw *TaskMiddleware[MWO], opts ...*MiddlewareOptions) {
	route.taskMws = append(route.taskMws, taskMiddlewareWithOptions{
		mw:   taskMw,
		opts: getFirstOpt(opts),
	})
}

func SetPatternLevelHTTPMiddleware[I any, O any](route *Route[I, O], httpMw HTTPMiddleware, opts ...*MiddlewareOptions) {
	route.httpMws = append(route.httpMws, httpMiddlewareWithOptions{
		mw:   httpMw,
		opts: getFirstOpt(opts),
	})
}

func getFirstOpt(opts []*MiddlewareOptions) *MiddlewareOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return nil
}

func SetGlobalNotFoundHTTPHandler(router *Router, httpHandler http.Handler) {
	router.notFoundHandler = httpHandler
}

type Route[I, O any] struct {
	genericsutil.ZeroHelper[I, O]

	router          *Router
	method          string
	originalPattern string

	httpMws []httpMiddlewareWithOptions
	taskMws []taskMiddlewareWithOptions

	handlerType     string
	userHTTPHandler http.Handler
	taskHandler     tasks.AnyTask

	needsTasksCtx bool
	compiledHTTP  atomic.Value
}

type AnyRoute interface {
	genericsutil.AnyZeroHelper

	getHandlerType() string
	getHTTPHandler() http.Handler
	getTaskHandler() tasks.AnyTask
	getHTTPMws() []httpMiddlewareWithOptions
	getTaskMws() []taskMiddlewareWithOptions
	getNeedsTasksCtx() bool
	OriginalPattern() string
	Method() string
	HTTPChain(rt *Router, mm *methodMatcher) http.Handler
}

func (route *Route[I, O]) getHandlerType() string                  { return route.handlerType }
func (route *Route[I, O]) getHTTPHandler() http.Handler            { return route.userHTTPHandler }
func (route *Route[I, O]) getTaskHandler() tasks.AnyTask           { return route.taskHandler }
func (route *Route[I, O]) getHTTPMws() []httpMiddlewareWithOptions { return route.httpMws }
func (route *Route[I, O]) getTaskMws() []taskMiddlewareWithOptions { return route.taskMws }
func (route *Route[I, O]) getNeedsTasksCtx() bool                  { return route.needsTasksCtx }
func (route *Route[I, O]) OriginalPattern() string                 { return route.originalPattern }
func (route *Route[I, O]) Method() string                          { return route.method }

func (r *Route[I, O]) HTTPChain(rt *Router, mm *methodMatcher) http.Handler {
	if h, ok := r.compiledHTTP.Load().(http.Handler); ok {
		return h
	}
	h := applyHTTPMiddlewares(r.getHTTPHandler(), r.httpMws, mm.httpMws, rt.httpMws)
	r.compiledHTTP.Store(h)
	return h
}

// TaskHandlers are used for JSON responses only, and they are intended to
// be particularly convenient for sending JSON. If you need to send a different
// content type, use a traditional http.Handler instead.
func RegisterTaskHandler[I any, O any](
	router *Router, method, pattern string, taskHandler *TaskHandler[I, O],
) *Route[I, O] {
	route := newRouteStruct[I, O](router, method, pattern)
	route.handlerType = "task"
	route.taskHandler = taskHandler
	mm := router.getOrCreateMethodMatcher(method)
	mm.reqDataGetters[pattern] = createReqDataGetter(route)
	router.registerRoute(route)
	return route
}

func RegisterHandlerFunc(router *Router, method, pattern string, httpHandlerFunc http.HandlerFunc) *Route[any, any] {
	return RegisterHandler(router, method, pattern, httpHandlerFunc)
}

func RegisterHandler(router *Router, method, pattern string, httpHandler http.Handler) *Route[any, any] {
	route := newRouteStruct[any, any](router, method, pattern)
	route.handlerType = "http"
	route.userHTTPHandler = httpHandler
	route.needsTasksCtx = reflectutil.ImplementsInterface(
		reflect.TypeOf(httpHandler), HandlerNeedsTasksCtxImplReflectType,
	)
	mm := router.getOrCreateMethodMatcher(method)
	mm.reqDataGetters[pattern] = createReqDataGetter(route)
	router.registerRoute(route)
	return route
}

type reqDataMarker interface {
	getInput() any
	getUnderlyingReqDataInstance() any
	Params() Params
	SplatValues() []string
	TasksCtx() *tasks.TasksCtx
	Request() *http.Request
	ResponseProxy() *response.Proxy
}

func (rd *ReqData[I]) getInput() any                     { return rd.input }
func (rd *ReqData[I]) getUnderlyingReqDataInstance() any { return rd }
func (rd *ReqData[I]) Params() Params                    { return rd.params }
func (rd *ReqData[I]) SplatValues() []string             { return rd.splatVals }
func (rd *ReqData[I]) TasksCtx() *tasks.TasksCtx         { return rd.tasksCtx }
func (rd *ReqData[I]) Request() *http.Request            { return rd.req }
func (rd *ReqData[I]) ResponseProxy() *response.Proxy    { return rd.responseProxy }
func (rd *ReqData[I]) Input() I                          { return rd.input }

type reqDataGetter interface {
	getReqData(r *http.Request, tasksCtx *tasks.TasksCtx, match *matcher.BestMatch) (reqDataMarker, error)
}

type reqDataGetterImpl[I any] func(*http.Request, *tasks.TasksCtx, *matcher.BestMatch) (*ReqData[I], error)

func (f reqDataGetterImpl[I]) getReqData(r *http.Request, tasksCtx *tasks.TasksCtx, m *matcher.BestMatch) (reqDataMarker, error) {
	return f(r, tasksCtx, m)
}

func GetTasksCtx(r *http.Request) *tasks.TasksCtx {
	if rd := requestStore.GetValueFromContext(r.Context()); rd != nil {
		return rd.tasksCtx
	}
	return nil
}

func GetParam(r *http.Request, key string) string {
	return GetParams(r)[key]
}

func GetParams(r *http.Request) Params {
	if rd := requestStore.GetValueFromContext(r.Context()); rd != nil {
		return rd.params
	}
	return emptyParams
}

func GetSplatValues(r *http.Request) []string {
	if rd := requestStore.GetValueFromContext(r.Context()); rd != nil {
		return rd.splatVals
	}
	return emptySplatValues
}

type findBestOutput struct {
	methodMatcher     *methodMatcher
	match             *matcher.BestMatch
	didMatch          bool
	headFellBackToGet bool
}

func (rt *Router) findBestMatcherAndMatch(method string, realPath string) *findBestOutput {
	isHead := method == http.MethodHead
	if isHead {
		if headMatcher, ok := rt.methodToMatcherMap[http.MethodHead]; ok {
			if match, found := headMatcher.matcher.FindBestMatch(realPath); found {
				return &findBestOutput{
					methodMatcher: headMatcher,
					match:         match,
					didMatch:      true,
				}
			}
		}
		method = http.MethodGet
	}
	methodMatcher, ok := rt.methodToMatcherMap[method]
	if !ok {
		return &findBestOutput{}
	}
	match, ok := methodMatcher.matcher.FindBestMatch(realPath)
	if !ok {
		return &findBestOutput{}
	}
	return &findBestOutput{
		methodMatcher:     methodMatcher,
		match:             match,
		didMatch:          true,
		headFellBackToGet: isHead,
	}
}

func treatGetAsHead(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	headRW := &headResponseWriter{
		ResponseWriter: w,
		header:         make(http.Header),
		statusCode:     http.StatusOK,
	}
	handler.ServeHTTP(headRW, r)
	for k, values := range headRW.header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(headRW.statusCode)
}

type headResponseWriter struct {
	http.ResponseWriter
	header     http.Header
	statusCode int
}

func (hw *headResponseWriter) Header() http.Header            { return hw.header }
func (hw *headResponseWriter) WriteHeader(statusCode int)     { hw.statusCode = statusCode }
func (hw *headResponseWriter) Write(data []byte) (int, error) { return len(data), nil }

func (rt *Router) hasAnyTaskMiddleware(methodMatcher *methodMatcher, route AnyRoute) bool {
	return len(route.getTaskMws()) > 0 ||
		len(methodMatcher.taskMws) > 0 ||
		len(rt.taskMws) > 0
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			muxLog.Error("Recovered from panic", "error", rec)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()

	pathToUse := r.URL.Path
	if rt.mountRoot != "" && strings.HasPrefix(pathToUse, rt.mountRoot) {
		pathToUse = "/" + pathToUse[len(rt.mountRoot):]
	}

	best := rt.findBestMatcherAndMatch(r.Method, pathToUse)
	if !best.didMatch {
		if rt.notFoundHandler != nil {
			rt.notFoundHandler.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	match := best.match
	mm := best.methodMatcher
	route := mm.routes[match.OriginalPattern()]

	// Fast path for pure HTTP handlers without task middleware
	if route.getHandlerType() == "http" &&
		!rt.hasAnyTaskMiddleware(mm, route) &&
		!rt.injectTasksCtx &&
		!route.getNeedsTasksCtx() {
		if len(match.Params) > 0 || len(match.SplatValues) > 0 {
			rd := &rdTransport{
				params:    match.Params,
				splatVals: match.SplatValues,
				req:       r,
			}
			r = requestStore.GetRequestWithContext(r, rd)
		}

		handler := route.HTTPChain(rt, mm)

		if best.headFellBackToGet {
			treatGetAsHead(handler, w, r)
		} else {
			handler.ServeHTTP(w, r)
		}
		return
	}

	// Slow path: create TasksCtx and full request data
	tasksCtx := tasks.NewTasksCtx(r.Context())

	rd := &rdTransport{
		params:        match.Params,
		splatVals:     match.SplatValues,
		tasksCtx:      tasksCtx,
		req:           r,
		responseProxy: response.NewProxy(),
	}
	r = requestStore.GetRequestWithContext(r, rd)

	reqGetter := mm.reqDataGetters[match.OriginalPattern()]
	reqData, err := reqGetter.getReqData(r, tasksCtx, match)
	if err != nil {
		if validate.IsValidationError(err) {
			muxLog.Error("Validation error", "error", err, "pattern", match.OriginalPattern())
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			muxLog.Error("Internal server error", "error", err, "pattern", match.OriginalPattern())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	var finalHandler http.Handler
	if route.getHandlerType() == "http" {
		finalHandler = route.HTTPChain(rt, mm)
	} else {
		finalHandler = rt.createTaskFinalHandler(route, reqData)
	}

	handlerWithMW := rt.runAppropriateMws(tasksCtx, reqData, mm, route, finalHandler)

	if best.headFellBackToGet {
		treatGetAsHead(handlerWithMW, w, r)
	} else {
		handlerWithMW.ServeHTTP(w, r)
	}
}

func applyHTTPMiddlewareWithOptions(mwWithOpts httpMiddlewareWithOptions, handler http.Handler) http.Handler {
	if mwWithOpts.opts != nil && mwWithOpts.opts.If != nil {
		originalHandler := handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mwWithOpts.opts.If(r) {
				originalHandler.ServeHTTP(w, r)
			} else {
				mwWithOpts.mw(originalHandler).ServeHTTP(w, r)
			}
		})
	}
	return mwWithOpts.mw(handler)
}

func applyHTTPMiddlewares(
	handler http.Handler,
	routeMws []httpMiddlewareWithOptions,
	methodMws []httpMiddlewareWithOptions,
	globalMws []httpMiddlewareWithOptions,
) http.Handler {
	// Apply in reverse order for proper nesting
	// Pattern-level middlewares (innermost)
	for i := len(routeMws) - 1; i >= 0; i-- {
		handler = applyHTTPMiddlewareWithOptions(routeMws[i], handler)
	}
	// Method-level middlewares
	for i := len(methodMws) - 1; i >= 0; i-- {
		handler = applyHTTPMiddlewareWithOptions(methodMws[i], handler)
	}
	// Global middlewares (outermost)
	for i := len(globalMws) - 1; i >= 0; i-- {
		handler = applyHTTPMiddlewareWithOptions(globalMws[i], handler)
	}
	return handler
}

type middlewareTaskCallable struct {
	taskToRun tasks.AnyTask
	input     *ReqData[None]
}

func (m *middlewareTaskCallable) Run(ctx *tasks.TasksCtx) error {
	_, err := m.taskToRun.Do(ctx, m.input)
	return err
}

func (m *middlewareTaskCallable) IsCallable() {}

func (rt *Router) gatherAllTaskMiddlewares(
	methodMatcher *methodMatcher, routeMarker AnyRoute,
) []taskMiddlewareWithOptions {
	taskMwsRoute := routeMarker.getTaskMws()

	if len(rt.taskMws) == 0 && len(methodMatcher.taskMws) == 0 && len(taskMwsRoute) == 0 {
		return nil
	}

	cap := len(taskMwsRoute) + len(methodMatcher.taskMws) + len(rt.taskMws)
	allTaskMws := make([]taskMiddlewareWithOptions, 0, cap)
	allTaskMws = append(allTaskMws, rt.taskMws...)
	allTaskMws = append(allTaskMws, methodMatcher.taskMws...)
	allTaskMws = append(allTaskMws, taskMwsRoute...)
	return allTaskMws
}

func (rt *Router) createTaskFinalHandler(route AnyRoute, reqDataMarker reqDataMarker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := response.New(w)
		taskHandler := route.getTaskHandler()
		inputData := reqDataMarker.getUnderlyingReqDataInstance()

		data, err := taskHandler.Do(reqDataMarker.TasksCtx(), inputData)
		if err != nil {
			muxLog.Error("Error executing task handler", "error", err, "pattern", route.OriginalPattern())
			res.InternalServerError()
			return
		}

		responseProxy := reqDataMarker.ResponseProxy()
		responseProxy.ApplyToResponseWriter(w, r)

		if responseProxy.IsError() || responseProxy.IsRedirect() {
			return
		}

		if reflectutil.ExcludingNoneGetIsNilOrUltimatelyPointsToNil(data) {
			muxLog.Warn(
				"Do not return nil values from task handlers unless: (i) the underlying type is an empty struct or pointer to an empty struct; or (ii) you are returning an error.",
				"pattern", route.OriginalPattern(),
			)
		}

		res.JSON(data)
	})
}

func (rt *Router) runAppropriateMws(
	tasksCtx *tasks.TasksCtx,
	reqDataMarker reqDataMarker,
	methodMatcher *methodMatcher,
	routeMarker AnyRoute,
	finalHandler http.Handler,
) http.Handler {
	var handlerWithHTTPMws http.Handler
	if routeMarker.getHandlerType() == "http" {
		handlerWithHTTPMws = finalHandler
	} else {
		handlerWithHTTPMws = applyHTTPMiddlewares(finalHandler, routeMarker.getHTTPMws(), methodMatcher.httpMws, rt.httpMws)
	}

	collected := rt.gatherAllTaskMiddlewares(methodMatcher, routeMarker)
	if len(collected) == 0 {
		return handlerWithHTTPMws
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callables := make([]tasks.Callable, 0, len(collected))
		reqDataInstances := make([]*ReqData[None], 0, len(collected))

		for _, taskWithOpts := range collected {
			if taskWithOpts.opts != nil && taskWithOpts.opts.If != nil && !taskWithOpts.opts.If(r) {
				continue
			}
			rdForMw := &ReqData[None]{
				params:        reqDataMarker.Params(),
				splatVals:     reqDataMarker.SplatValues(),
				tasksCtx:      tasksCtx,
				input:         None{},
				req:           r,
				responseProxy: response.NewProxy(),
			}
			reqDataInstances = append(reqDataInstances, rdForMw)
			callables = append(callables, &middlewareTaskCallable{
				taskToRun: taskWithOpts.mw,
				input:     rdForMw,
			})
		}

		if err := tasks.Go(tasksCtx, callables...); err != nil {
			muxLog.Error("Error during parallel middleware execution", "error", err)
		}

		proxies := make([]*response.Proxy, len(reqDataInstances))
		for i, rdInst := range reqDataInstances {
			proxies[i] = rdInst.ResponseProxy()
		}
		merged := response.MergeProxyResponses(proxies...)
		merged.ApplyToResponseWriter(w, r)
		if merged.IsError() || merged.IsRedirect() {
			return
		}

		handlerWithHTTPMws.ServeHTTP(w, r)
	})
}

func newRouteStruct[I any, O any](router *Router, method, originalPattern string) *Route[I, O] {
	return &Route[I, O]{
		router: router, method: method, originalPattern: originalPattern,
		httpMws: emptyHTTPMws, taskMws: emptyTaskMws,
	}
}

func (rt *Router) registerRoute(route AnyRoute) {
	methodMatcher := rt.getOrCreateMethodMatcher(route.Method())
	methodMatcher.matcher.RegisterPattern(route.OriginalPattern())
	methodMatcher.routes[route.OriginalPattern()] = route
	rt.allRoutes = append(rt.allRoutes, route)
}

func createReqDataGetter[I any, O any](route *Route[I, O]) reqDataGetter {
	return reqDataGetterImpl[I](
		func(r *http.Request, tasksCtx *tasks.TasksCtx, match *matcher.BestMatch) (*ReqData[I], error) {
			reqData := new(ReqData[I])
			reqData.params = match.Params
			reqData.splatVals = match.SplatValues
			reqData.tasksCtx = tasksCtx
			reqData.req = r
			reqData.responseProxy = response.NewProxy()

			inputPtr := route.IPtr()
			if route.router.marshalInput != nil && !genericsutil.IsNone(route.I()) {
				if err := route.router.marshalInput(reqData.Request(), inputPtr); err != nil {
					return nil, err
				}
			}
			reqData.input = *(inputPtr.(*I))
			return reqData, nil
		},
	)
}

func (rt *Router) getOrCreateMethodMatcher(method string) *methodMatcher {
	if mm, ok := rt.methodToMatcherMap[method]; ok {
		return mm
	}

	mm := &methodMatcher{
		matcher:        matcher.New(rt.matcherOpts),
		routes:         make(map[string]AnyRoute),
		reqDataGetters: make(map[string]reqDataGetter),
		httpMws:        emptyHTTPMws,
		taskMws:        emptyTaskMws,
	}
	rt.methodToMatcherMap[method] = mm
	return mm
}
