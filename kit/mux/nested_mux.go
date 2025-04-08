package mux

import (
	"fmt"
	"net/http"

	"github.com/sjc5/river/kit/genericsutil"
	"github.com/sjc5/river/kit/matcher"
	"github.com/sjc5/river/kit/opt"
	"github.com/sjc5/river/kit/response"
	"github.com/sjc5/river/kit/tasks"
)

type NestedReqData = ReqData[None]

/////////////////////////////////////////////////////////////////////
/////// CORE ROUTER STRUCTURE
/////////////////////////////////////////////////////////////////////

// Always a GET / no input parsing / all tasks

type NestedRouter struct {
	_tasks_registry *tasks.Registry
	_matcher        *matcher.Matcher
	_routes         map[string]AnyNestedRoute
}

func (nr *NestedRouter) AllRoutes() map[string]AnyNestedRoute {
	return nr._routes
}

func (nr *NestedRouter) TasksRegistry() *tasks.Registry {
	return nr._tasks_registry
}

func (nr *NestedRouter) IsRegistered(pattern string) bool {
	_, exists := nr._routes[pattern]
	return exists
}

func (nr *NestedRouter) GetExplicitIndexSegment() string {
	return nr._matcher.GetExplicitIndexSegment()
}

/////////////////////////////////////////////////////////////////////
/////// NEW ROUTER
/////////////////////////////////////////////////////////////////////

type NestedOptions struct {
	TasksRegistry          *tasks.Registry
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

	if opts.TasksRegistry == nil {
		panic("tasks registry is required for nested router")
	}

	_matcher_opts.DynamicParamPrefixRune = opt.Resolve(opts, opts.DynamicParamPrefixRune, ':')
	_matcher_opts.SplatSegmentRune = opt.Resolve(opts, opts.SplatSegmentRune, '*')
	_matcher_opts.ExplicitIndexSegment = opt.Resolve(opts, opts.ExplicitIndexSegment, "")

	return &NestedRouter{
		_tasks_registry: opts.TasksRegistry,
		_matcher:        matcher.New(_matcher_opts),
		_routes:         make(map[string]AnyNestedRoute),
	}
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (CORE)
/////////////////////////////////////////////////////////////////////

type NestedRoute[O any] struct {
	genericsutil.ZeroHelper[None, O]

	_router  *NestedRouter
	_pattern string

	_task_handler tasks.AnyRegisteredTask
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED PATTERNS (COBWEBS)
/////////////////////////////////////////////////////////////////////

type AnyNestedRoute interface {
	genericsutil.AnyZeroHelper
	_get_task_handler() tasks.AnyRegisteredTask
	Pattern() string
}

func (route *NestedRoute[O]) _get_task_handler() tasks.AnyRegisteredTask { return route._task_handler }
func (route *NestedRoute[O]) Pattern() string                            { return route._pattern }

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
	_pattern string
	_data    any
	_err     error
}

func (ntr *NestedTasksResult) Pattern() string { return ntr._pattern }
func (ntr *NestedTasksResult) OK() bool        { return ntr._err == nil }
func (ntr *NestedTasksResult) Data() any       { return ntr._data }
func (ntr *NestedTasksResult) Err() error      { return ntr._err }

type NestedTasksResults struct {
	Params          Params
	SplatValues     []string
	Map             map[string]*NestedTasksResult
	Slice           []*NestedTasksResult
	ResponseProxies []*response.Proxy
	_task_indices   map[int]int // Maps match index to task index
}

func (ntr *NestedTasksResults) GetHasTaskHandler(i int) bool {
	_, ok := ntr._task_indices[i]
	return ok
}

// Second return value (bool) indicates matches found
func FindNestedMatches(nestedRouter *NestedRouter, r *http.Request) (*matcher.FindNestedMatchesResults, bool) {
	return nestedRouter._matcher.FindNestedMatches(r.URL.Path)
}

// Second return value (bool) indicates matches found, not success of tasks run
func FindNestedMatchesAndRunTasks(nestedRouter *NestedRouter, tasksCtx *tasks.TasksCtx, r *http.Request) (*NestedTasksResults, bool) {
	_results, ok := FindNestedMatches(nestedRouter, r)
	if !ok {
		return nil, false
	}

	return RunNestedTasks(nestedRouter, tasksCtx, r, _results), true
}

func RunNestedTasks(
	nestedRouter *NestedRouter,
	tasksCtx *tasks.TasksCtx,
	r *http.Request,
	findNestedMatchesResults *matcher.FindNestedMatchesResults,
) *NestedTasksResults {
	matches := findNestedMatchesResults.Matches

	if len(matches) == 0 {
		return nil
	}

	_results := new(NestedTasksResults)
	_results.Params = findNestedMatchesResults.Params
	_results.SplatValues = findNestedMatchesResults.SplatValues

	// Initialize result containers up front
	_results.Map = make(map[string]*NestedTasksResult, len(matches))
	_results.Slice = make([]*NestedTasksResult, len(matches))

	// First, identify which matches have tasks that need to be run
	_tasks_with_input := make([]tasks.AnyPreparedTask, 0, len(matches))
	_results.ResponseProxies = make([]*response.Proxy, 0, len(matches))
	_results._task_indices = make(map[int]int) // Maps match index to task index

	for i, _match := range matches {
		_response_proxy := response.NewProxy()
		_results.ResponseProxies = append(_results.ResponseProxies, _response_proxy)

		_nested_route_marker, routeExists := nestedRouter._routes[_match.OriginalPattern()]

		// Create result object regardless of whether a task exists
		_res := &NestedTasksResult{_pattern: _match.OriginalPattern()}
		_results.Map[_match.OriginalPattern()] = _res
		_results.Slice[i] = _res

		// Skip task preparation if route doesn't exist or has no task handler
		if !routeExists {
			continue
		}

		_task := _nested_route_marker._get_task_handler()
		if _task == nil {
			// This means a user registered a pattern but didn't provide a task handler.
			// In this case, just continue.
			continue
		}

		_rd := &ReqData[None]{
			_params:         findNestedMatchesResults.Params,
			_splat_vals:     findNestedMatchesResults.SplatValues,
			_tasks_ctx:      tasksCtx,
			_input:          None{},
			_response_proxy: _response_proxy,
		}

		_tasks_with_input = append(_tasks_with_input, tasks.PrepAny(tasksCtx, _task, _rd))
		_results._task_indices[i] = len(_tasks_with_input) - 1 // Store the mapping between match index and task index
	}

	// Only run parallelPreload if we have tasks to run
	if len(_tasks_with_input) > 0 {
		tasksCtx.ParallelPreload(_tasks_with_input...)
	}

	// Process task results for matches that had tasks
	for matchIdx, taskIdx := range _results._task_indices {
		_res := _results.Slice[matchIdx]
		_data, err := _tasks_with_input[taskIdx].GetAny()
		_res._data = _data
		_res._err = err
	}

	return _results
}

/////////////////////////////////////////////////////////////////////
/////// INTERNAL HELPERS
/////////////////////////////////////////////////////////////////////

func _new_nested_route_struct[O any](_router *NestedRouter, _pattern string) *NestedRoute[O] {
	return &NestedRoute[O]{_router: _router, _pattern: _pattern}
}

func _must_register_nested_route[O any](_route *NestedRoute[O]) {
	_matcher := _route._router._matcher
	_matcher.RegisterPattern(_route._pattern)
	_, _already_exists := _route._router._routes[_route._pattern]
	if _already_exists {
		panic(fmt.Sprintf("pattern already registered: %s", _route._pattern))
	}
	_route._router._routes[_route._pattern] = _route
}
