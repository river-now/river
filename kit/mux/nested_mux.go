package mux

import (
	"fmt"
	"net/http"

	"github.com/river-now/river/kit/genericsutil"
	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/opt"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/tasks"
)

type NestedReqData = ReqData[None]

/////////////////////////////////////////////////////////////////////
/////// CORE NESTED ROUTER STRUCTURE
/////////////////////////////////////////////////////////////////////

// Always a GET / no input parsing / all tasks

type NestedRouter struct {
	matcher *matcher.Matcher
	routes  map[string]AnyNestedRoute
}

func (nr *NestedRouter) AllRoutes() map[string]AnyNestedRoute {
	return nr.routes
}

func (nr *NestedRouter) IsRegistered(originalPattern string) bool {
	_, exists := nr.routes[originalPattern]
	return exists
}

func (nr *NestedRouter) GetExplicitIndexSegment() string {
	return nr.matcher.GetExplicitIndexSegment()
}

func (nr *NestedRouter) GetDynamicParamPrefixRune() rune {
	return nr.matcher.GetDynamicParamPrefixRune()
}

func (nr *NestedRouter) GetSplatSegmentRune() rune {
	return nr.matcher.GetSplatSegmentRune()
}

/////////////////////////////////////////////////////////////////////
/////// NEW NESTED ROUTER
/////////////////////////////////////////////////////////////////////

type NestedOptions struct {
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.

	// Optional. Defaults to empty string (trailing slash in your patterns).
	// You can set it to something like "_index" to make it explicit.
	ExplicitIndexSegment string
}

func NewNestedRouter(opts *NestedOptions) *NestedRouter {
	matcherOpts := new(matcher.Options)

	if opts == nil {
		opts = new(NestedOptions)
	}

	matcherOpts.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	matcherOpts.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')
	matcherOpts.ExplicitIndexSegment = opt.Resolve(opts, opts.ExplicitIndexSegment, "")

	return &NestedRouter{
		matcher: matcher.New(matcherOpts),
		routes:  make(map[string]AnyNestedRoute),
	}
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (CORE)
/////////////////////////////////////////////////////////////////////

type NestedRoute[O any] struct {
	genericsutil.ZeroHelper[None, O]

	router          *NestedRouter
	originalPattern string
	taskHandler     tasks.AnyTask
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (INTERFACES)
/////////////////////////////////////////////////////////////////////

type AnyNestedRoute interface {
	genericsutil.AnyZeroHelper
	getTaskHandler() tasks.AnyTask
	OriginalPattern() string
}

func (route *NestedRoute[O]) getTaskHandler() tasks.AnyTask { return route.taskHandler }
func (route *NestedRoute[O]) OriginalPattern() string       { return route.originalPattern }

/////////////////////////////////////////////////////////////////////
/////// CORE PATTERN REGISTRATION FUNCTIONS
/////////////////////////////////////////////////////////////////////

func RegisterNestedTaskHandler[O any](
	router *NestedRouter, pattern string, taskHandler *TaskHandler[None, O],
) *NestedRoute[O] {
	route := &NestedRoute[O]{
		router:          router,
		originalPattern: pattern,
		taskHandler:     taskHandler,
	}
	mustRegisterNestedRoute(route)
	return route
}

func RegisterNestedPatternWithoutHandler(router *NestedRouter, pattern string) {
	route := &NestedRoute[None]{
		router:          router,
		originalPattern: pattern,
		taskHandler:     nil,
	}
	mustRegisterNestedRoute(route)
}

/////////////////////////////////////////////////////////////////////
/////// RUN NESTED TASKS
/////////////////////////////////////////////////////////////////////

type NestedTasksResult struct {
	pattern string
	data    any
	err     error
	ranTask bool
}

func (ntr *NestedTasksResult) Pattern() string { return ntr.pattern }
func (ntr *NestedTasksResult) OK() bool        { return ntr.err == nil }
func (ntr *NestedTasksResult) Data() any       { return ntr.data }
func (ntr *NestedTasksResult) Err() error      { return ntr.err }
func (ntr *NestedTasksResult) RanTask() bool   { return ntr.ranTask }

type NestedTasksResults struct {
	Params          Params
	SplatValues     []string
	Map             map[string]*NestedTasksResult
	Slice           []*NestedTasksResult
	ResponseProxies []*response.Proxy
}

func (ntr *NestedTasksResults) GetHasTaskHandler(i int) bool {
	if i < 0 || i >= len(ntr.Slice) {
		return false
	}
	return ntr.Slice[i].ranTask
}

func FindNestedMatches(nestedRouter *NestedRouter, r *http.Request) (*matcher.FindNestedMatchesResults, bool) {
	return nestedRouter.matcher.FindNestedMatches(r.URL.Path)
}

func FindNestedMatchesAndRunTasks(nestedRouter *NestedRouter, tasksCtx *tasks.Context, r *http.Request) (*NestedTasksResults, bool) {
	findResults, ok := FindNestedMatches(nestedRouter, r)
	if !ok {
		return nil, false
	}
	return RunNestedTasks(nestedRouter, tasksCtx, r, findResults), true
}

// Helper struct to bridge AnyTask with tasks.Go Callable
type nestedTaskCallable struct {
	taskToRun  tasks.AnyTask  // The actual task
	input      *ReqData[None] // The input for this task
	resultDest *any           // Pointer to store the data
	errorDest  *error         // Pointer to store the error
}

func (nc *nestedTaskCallable) Run(ctx *tasks.Context) error {
	data, err := nc.taskToRun.Do(ctx, nc.input)
	*nc.resultDest = data
	*nc.errorDest = err
	return err
}

func (nc *nestedTaskCallable) IsCallable() {}

func RunNestedTasks(
	nestedRouter *NestedRouter,
	tasksCtx *tasks.Context,
	r *http.Request,
	findNestedMatchesResults *matcher.FindNestedMatchesResults,
) *NestedTasksResults {
	matches := findNestedMatchesResults.Matches

	if len(matches) == 0 {
		return nil
	}

	results := &NestedTasksResults{
		Params:      findNestedMatchesResults.Params,
		SplatValues: findNestedMatchesResults.SplatValues,
	}

	// Initialize result containers up front
	results.Map = make(map[string]*NestedTasksResult, len(matches))
	results.Slice = make([]*NestedTasksResult, len(matches))
	results.ResponseProxies = make([]*response.Proxy, len(matches))

	// Pre-allocate callables with estimated capacity
	callables := make([]tasks.Callable, 0, len(matches))

	for i, match := range matches {
		pattern := match.OriginalPattern()

		// Initialize NestedTasksResult for every match
		currentResult := &NestedTasksResult{pattern: pattern}
		results.Map[pattern] = currentResult
		results.Slice[i] = currentResult

		// Create response proxy for every match (maintains index alignment)
		currentResultProxy := response.NewProxy()
		results.ResponseProxies[i] = currentResultProxy

		// Check if route exists first to avoid nil panic
		nestedRouteMarker, routeExists := nestedRouter.routes[pattern]
		if !routeExists {
			continue
		}

		// Now safe to get task handler
		taskToRun := nestedRouteMarker.getTaskHandler()
		if taskToRun == nil {
			// This means a user registered a pattern but didn't provide a task handler.
			// In this case, just continue.
			continue
		}

		currentResult.ranTask = true

		// Create request data with all fields properly initialized
		reqDataForTask := &ReqData[None]{
			params:        results.Params,
			splatVals:     results.SplatValues,
			tasksCtx:      tasksCtx,
			input:         None{}, // Important: initialize the input field
			req:           r,
			responseProxy: currentResultProxy,
		}

		// Create a callable that will store the result directly into the currentResult
		callables = append(callables, &nestedTaskCallable{
			taskToRun:  taskToRun,
			input:      reqDataForTask,
			resultDest: &currentResult.data,
			errorDest:  &currentResult.err,
		})
	}

	// Run all prepared callables in parallel
	if len(callables) > 0 {
		if err := tasks.Go(tasksCtx, callables...); err != nil {
			muxLog.Error("tasks.Go reported an error during nested task execution", "error", err)
		}
	}

	return results
}

/////////////////////////////////////////////////////////////////////
/////// INTERNAL HELPERS
/////////////////////////////////////////////////////////////////////

func mustRegisterNestedRoute[O any](route *NestedRoute[O]) {
	if route.router.IsRegistered(route.originalPattern) {
		panic(fmt.Sprintf("Pattern '%s' is already registered in NestedRouter. Perhaps you're unintentionally registering it twice?", route.originalPattern))
	}

	route.router.matcher.RegisterPattern(route.originalPattern)
	route.router.routes[route.originalPattern] = route
}
