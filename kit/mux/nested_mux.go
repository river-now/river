package mux

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/river-now/river/kit/genericsutil"
	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/opt"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/tasks"
)

// Pre-allocated structures to reduce allocations
var (
	noneInstance = None{}
)

type NestedReqData = ReqData[None]

// Always a GET / no input parsing / all tasks

type compiledNestedRoute struct {
	pattern     string
	taskHandler tasks.AnyTask
	hasHandler  bool
}

type NestedRouter struct {
	matcher        *matcher.Matcher
	routes         map[string]AnyNestedRoute
	compiledRoutes map[string]*compiledNestedRoute
	mu             sync.RWMutex
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
		matcher:        matcher.New(matcherOpts),
		routes:         make(map[string]AnyNestedRoute),
		compiledRoutes: make(map[string]*compiledNestedRoute),
	}
}

type NestedRoute[O any] struct {
	genericsutil.ZeroHelper[None, O]

	router          *NestedRouter
	originalPattern string
	taskHandler     tasks.AnyTask
}

type AnyNestedRoute interface {
	genericsutil.AnyZeroHelper
	getTaskHandler() tasks.AnyTask
	OriginalPattern() string
}

func (route *NestedRoute[O]) getTaskHandler() tasks.AnyTask { return route.taskHandler }
func (route *NestedRoute[O]) OriginalPattern() string       { return route.originalPattern }

func RegisterNestedTaskHandler[O any](
	router *NestedRouter, pattern string, taskHandler *TaskHandler[None, O],
) *NestedRoute[O] {
	route := &NestedRoute[O]{
		router:          router,
		originalPattern: pattern,
		taskHandler:     taskHandler,
	}
	mustRegisterNestedRoute(route)

	// Pre-compile route
	router.mu.Lock()
	router.compiledRoutes[pattern] = &compiledNestedRoute{
		pattern:     pattern,
		taskHandler: taskHandler,
		hasHandler:  true,
	}
	router.mu.Unlock()

	return route
}

func RegisterNestedPatternWithoutHandler(router *NestedRouter, pattern string) {
	route := &NestedRoute[None]{
		router:          router,
		originalPattern: pattern,
		taskHandler:     nil,
	}
	mustRegisterNestedRoute(route)

	// Pre-compile route
	router.mu.Lock()
	router.compiledRoutes[pattern] = &compiledNestedRoute{
		pattern:     pattern,
		taskHandler: nil,
		hasHandler:  false,
	}
	router.mu.Unlock()
}

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

func FindNestedMatchesAndRunTasks(nestedRouter *NestedRouter, r *http.Request) (*NestedTasksResults, bool) {
	findResults, ok := FindNestedMatches(nestedRouter, r)
	if !ok {
		return nil, false
	}
	return RunNestedTasks(nestedRouter, r, findResults), true
}

// Optimized task callable that embeds all needed data
type nestedTaskCallable struct {
	taskHandler tasks.AnyTask
	reqData     *ReqData[None]
	result      *NestedTasksResult
}

func (nc *nestedTaskCallable) Run(ctx *tasks.TasksCtx) error {
	data, err := nc.taskHandler.Do(ctx, nc.reqData)
	nc.result.data = data
	nc.result.err = err
	return err
}

func (nc *nestedTaskCallable) IsCallable() {}

func RunNestedTasks(
	nestedRouter *NestedRouter,
	r *http.Request,
	findNestedMatchesResults *matcher.FindNestedMatchesResults,
) *NestedTasksResults {
	tasksCtx := GetTasksCtx(r)
	if tasksCtx == nil {
		muxLog.Error("No TasksCtx found in request for RunNestedTasks")
		return nil
	}

	matches := findNestedMatchesResults.Matches
	if len(matches) == 0 {
		return nil
	}

	// Pre-allocate results structure
	numMatches := len(matches)
	results := &NestedTasksResults{
		Params:          findNestedMatchesResults.Params,
		SplatValues:     findNestedMatchesResults.SplatValues,
		Map:             make(map[string]*NestedTasksResult, numMatches),
		Slice:           make([]*NestedTasksResult, numMatches),
		ResponseProxies: make([]*response.Proxy, numMatches),
	}

	// First pass: create results and count tasks
	taskCount := 0
	nestedRouter.mu.RLock()
	for i, match := range matches {
		pattern := match.OriginalPattern()

		// Create result for every match
		result := &NestedTasksResult{pattern: pattern}
		results.Map[pattern] = result
		results.Slice[i] = result

		// Check if we have a compiled route with handler
		if compiled, exists := nestedRouter.compiledRoutes[pattern]; exists && compiled.hasHandler {
			result.ranTask = true
			taskCount++
		}
	}
	nestedRouter.mu.RUnlock()

	// If no tasks to run, create empty proxies and return
	if taskCount == 0 {
		for i := range results.ResponseProxies {
			results.ResponseProxies[i] = response.NewProxy()
		}
		return results
	}

	// Pre-allocate callables
	callables := make([]tasks.Callable, 0, taskCount)

	// Second pass: setup tasks
	nestedRouter.mu.RLock()
	for i, match := range matches {
		pattern := match.OriginalPattern()
		compiled, exists := nestedRouter.compiledRoutes[pattern]

		if !exists || !compiled.hasHandler {
			// Create empty proxy for non-task routes
			results.ResponseProxies[i] = response.NewProxy()
			continue
		}

		// Create response proxy
		proxy := response.NewProxy()
		results.ResponseProxies[i] = proxy

		// Create ReqData directly with fields
		reqData := &ReqData[None]{
			params:        results.Params,
			splatVals:     results.SplatValues,
			tasksCtx:      tasksCtx,
			input:         noneInstance,
			req:           r,
			responseProxy: proxy,
		}

		// Create callable
		callable := &nestedTaskCallable{
			taskHandler: compiled.taskHandler,
			reqData:     reqData,
			result:      results.Slice[i],
		}

		callables = append(callables, callable)
	}
	nestedRouter.mu.RUnlock()

	// Execute all tasks in parallel
	if err := tasks.Go(tasksCtx, callables...); err != nil {
		muxLog.Error("tasks.Go reported an error during nested task execution", "error", err)
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
