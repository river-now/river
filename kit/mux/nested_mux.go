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
/////// CORE ROUTER STRUCTURE
/////////////////////////////////////////////////////////////////////

// Always a GET / no input parsing / all tasks

type NestedRouter struct {
	_matcher *matcher.Matcher
	_routes  map[string]AnyNestedRoute
}

func (nr *NestedRouter) AllRoutes() map[string]AnyNestedRoute {
	return nr._routes
}

func (nr *NestedRouter) IsRegistered(originalPattern string) bool {
	_, exists := nr._routes[originalPattern]
	return exists
}

func (nr *NestedRouter) GetExplicitIndexSegment() string {
	return nr._matcher.GetExplicitIndexSegment()
}
func (nr *NestedRouter) GetDynamicParamPrefixRune() rune {
	return nr._matcher.GetDynamicParamPrefixRune()
}
func (nr *NestedRouter) GetSplatSegmentRune() rune {
	return nr._matcher.GetSplatSegmentRune()
}

/////////////////////////////////////////////////////////////////////
/////// NEW ROUTER
/////////////////////////////////////////////////////////////////////

type NestedOptions struct {
	DynamicParamPrefixRune rune // Optional. Defaults to ':'.
	SplatSegmentRune       rune // Optional. Defaults to '*'.

	// Optional. Defaults to empty string (trailing slash in your patterns).
	// You can set it to something like "_index" to make it explicit.
	ExplicitIndexSegment string
}

func NewNestedRouter(opts *NestedOptions) *NestedRouter {
	_matcher_opts := new(matcher.Options)

	if opts == nil {
		opts = new(NestedOptions)
	}

	_matcher_opts.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	_matcher_opts.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')
	_matcher_opts.ExplicitIndexSegment = opt.Resolve(opts, opts.ExplicitIndexSegment, "")

	return &NestedRouter{
		_matcher: matcher.New(_matcher_opts),
		_routes:  make(map[string]AnyNestedRoute),
	}
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (CORE)
/////////////////////////////////////////////////////////////////////

type NestedRoute[O any] struct {
	genericsutil.ZeroHelper[None, O]

	_router           *NestedRouter
	_original_pattern string

	_task_handler tasks.AnyTask
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (COBWEBS)
/////////////////////////////////////////////////////////////////////

type AnyNestedRoute interface {
	genericsutil.AnyZeroHelper
	_get_task_handler() tasks.AnyTask
	OriginalPattern() string
}

func (route *NestedRoute[O]) _get_task_handler() tasks.AnyTask { return route._task_handler }
func (route *NestedRoute[O]) OriginalPattern() string          { return route._original_pattern }

/////////////////////////////////////////////////////////////////////
/////// CORE PATTERN REGISTRATION FUNCTIONS
/////////////////////////////////////////////////////////////////////

func RegisterNestedTaskHandler[O any](
	router *NestedRouter, pattern string, taskHandler *TaskHandler[None, O],
) *NestedRoute[O] {
	_route := _new_nested_route_struct[O](router, pattern)
	_route._task_handler = taskHandler
	_must_register_nested_route(_route)
	return _route
}

func RegisterNestedPatternWithoutHandler(router *NestedRouter, pattern string) {
	_route := _new_nested_route_struct[None](router, pattern)
	_must_register_nested_route(_route)
}

/////////////////////////////////////////////////////////////////////
/////// RUN NESTED TASKS
/////////////////////////////////////////////////////////////////////

type NestedTasksResult struct {
	_pattern  string
	_data     any
	_err      error
	_ran_task bool
}

func (ntr *NestedTasksResult) Pattern() string { return ntr._pattern }
func (ntr *NestedTasksResult) OK() bool        { return ntr._err == nil }
func (ntr *NestedTasksResult) Data() any       { return ntr._data }
func (ntr *NestedTasksResult) Err() error      { return ntr._err }
func (ntr *NestedTasksResult) RanTask() bool   { return ntr._ran_task }

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
	return ntr.Slice[i]._ran_task
}

func FindNestedMatches(nestedRouter *NestedRouter, r *http.Request) (*matcher.FindNestedMatchesResults, bool) {
	return nestedRouter._matcher.FindNestedMatches(r.URL.Path)
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
	ctx        *tasks.Context // The tasks.Context for this execution
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

	results := new(NestedTasksResults)
	results.Params = findNestedMatchesResults.Params
	results.SplatValues = findNestedMatchesResults.SplatValues

	// Initialize result containers up front
	results.Map = make(map[string]*NestedTasksResult, len(matches))
	results.Slice = make([]*NestedTasksResult, len(matches))

	// First, identify which matches have tasks that need to be run
	results.ResponseProxies = make([]*response.Proxy, len(matches))

	callables := make([]tasks.Callable, 0, len(matches))

	for i, match := range matches {
		// Initialize NestedTasksResult for every match
		currentResult := &NestedTasksResult{_pattern: match.OriginalPattern()}
		results.Map[match.OriginalPattern()] = currentResult
		results.Slice[i] = currentResult
		currentResultProxy := response.NewProxy()
		results.ResponseProxies[i] = currentResultProxy

		nestedRouteMarker, routeExists := nestedRouter._routes[match.OriginalPattern()]
		if !routeExists {
			continue
		}

		taskToRun := nestedRouteMarker._get_task_handler()
		if taskToRun == nil {
			// This means a user registered a pattern but didn't provide a task handler.
			// In this case, just continue.
			continue
		}

		currentResult._ran_task = true

		reqDataForTask := &ReqData[None]{
			_params:         results.Params,
			_splat_vals:     results.SplatValues,
			_tasks_ctx:      tasksCtx,
			_input:          None{},
			_req:            r,
			_response_proxy: currentResultProxy,
		}

		// Create a callable that will store the result directly into the currentResult
		callables = append(callables, &nestedTaskCallable{
			ctx:        tasksCtx,
			taskToRun:  taskToRun,
			input:      reqDataForTask,
			resultDest: &currentResult._data,
			errorDest:  &currentResult._err,
		})
	}

	// Run all prepared callables in parallel
	if len(callables) > 0 {
		if err := tasks.Go(tasksCtx, callables...); err != nil {
			fmt.Printf("tasks.Go reported an error during nested task execution: %v\n", err)
		}
	}

	return results
}

/////////////////////////////////////////////////////////////////////
/////// INTERNAL HELPERS
/////////////////////////////////////////////////////////////////////

func _new_nested_route_struct[O any](_router *NestedRouter, _original_pattern string) *NestedRoute[O] {
	return &NestedRoute[O]{_router: _router, _original_pattern: _original_pattern}
}

func _must_register_nested_route[O any](_route *NestedRoute[O]) {
	_matcher := _route._router._matcher
	_matcher.RegisterPattern(_route._original_pattern)
	_, _already_exists := _route._router._routes[_route._original_pattern]
	if _already_exists {
		panic(fmt.Sprintf("Pattern '%s' is already registered in NestedRouter. Perhaps you're unintentionally registering it twice?", _route._original_pattern))
	}
	_route._router._routes[_route._original_pattern] = _route
}
