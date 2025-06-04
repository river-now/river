package tasks

import (
	"context"
	"net/http"
	"sync"

	"github.com/river-now/river/kit/genericsutil"
	"golang.org/x/sync/errgroup"
)

/////////////////////////////////////////////////////////////////////
/////// ARGS
/////////////////////////////////////////////////////////////////////

type Arg[I any] struct {
	Input I
	*TasksCtx
}

type ArgNoInput = Arg[genericsutil.None]

/////////////////////////////////////////////////////////////////////
/////// REGISTERED TASKS
/////////////////////////////////////////////////////////////////////

type AnyRegisteredTask interface {
	getID() int
	execute(c *TasksCtx, input any) (any, error)
}

type ioFunc[I any, O any] = func(c *Arg[I]) (O, error)

type RegisteredTask[I any, O any] struct {
	fn ioFunc[I, O]
	id int
}

func (task *RegisteredTask[I, O]) getID() int { return task.id }

func (task *RegisteredTask[I, O]) execute(c *TasksCtx, input any) (any, error) {
	return task.fn(&Arg[I]{
		TasksCtx: c,
		Input:    genericsutil.AssertOrZero[I](input),
	})
}

func Register[I any, O any](tr *Registry, f ioFunc[I, O]) *RegisteredTask[I, O] {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	task := &RegisteredTask[I, O]{
		fn: f,
		id: tr.count,
	}
	tr.count++
	return task
}

/////////////////////////////////////////////////////////////////////
/////// TASKS REGISTRY
/////////////////////////////////////////////////////////////////////

type Registry struct {
	key   string
	count int
	mu    sync.Mutex
}

func (tr *Registry) NewCtxFromNativeContext(parentContext context.Context) *TasksCtx {
	return newTasksCtx(parentContext, tr)
}

func (tr *Registry) NewCtxFromRequest(r *http.Request) *TasksCtx {
	return newTasksCtx(r.Context(), tr)
}

func NewRegistry(key string) *Registry {
	return &Registry{key: key}
}

/////////////////////////////////////////////////////////////////////
/////// CTX
/////////////////////////////////////////////////////////////////////

type result struct {
	once sync.Once
	data any
	err  error
}

type TasksCtx struct {
	mu       sync.Mutex
	registry *Registry
	results  map[AnyRegisteredTask]*result

	context context.Context
	cancel  context.CancelFunc
}

type tasksCtxContextKeyType string

var tasksCtxContextKey tasksCtxContextKeyType = "_tasks_ctx_"

func (tr *Registry) toContextKey() tasksCtxContextKeyType {
	return tasksCtxContextKey + tasksCtxContextKeyType(tr.key)
}

func (tr *Registry) GetCtxFromRequest(r *http.Request) *TasksCtx {
	_tasks_ctx, ok := r.Context().Value(tr.toContextKey()).(*TasksCtx)
	if !ok {
		return nil
	}
	return _tasks_ctx
}

func (tr *Registry) MustGetCtxFromRequest(r *http.Request) *TasksCtx {
	_tasks_ctx, ok := r.Context().Value(tr.toContextKey()).(*TasksCtx)
	if !ok {
		panic("tasks context not found in request")
	}
	return _tasks_ctx
}

func (tr *Registry) GetRequestWithCtxIfNeeded(r *http.Request) *http.Request {
	if tr.GetCtxFromRequest(r) != nil {
		return nil
	}
	tasksCtx := tr.NewCtxFromRequest(r)
	contextWithValue := context.WithValue(r.Context(), tr.toContextKey(), tasksCtx)
	return r.WithContext(contextWithValue)
}

func (tr *Registry) AddTasksCtxToRequestMw() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newR := tr.GetRequestWithCtxIfNeeded(r)
			if newR == nil {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, newR)
		})
	}
}

func newTasksCtx(parentContext context.Context, tr *Registry) *TasksCtx {
	contextWithCancel, cancel := context.WithCancel(parentContext)

	return &TasksCtx{
		registry: tr,
		context:  contextWithCancel,
		cancel:   cancel,
		results:  make(map[AnyRegisteredTask]*result),
	}
}

func (c *TasksCtx) NativeContext() context.Context {
	return c.context
}

func (c *TasksCtx) CancelNativeContext() {
	c.cancel()
}

func (c *TasksCtx) getResult(task AnyRegisteredTask) *result {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.results[task]; ok {
		return r
	}
	r := &result{}
	c.results[task] = r
	return r
}

/////////////////////////////////////////////////////////////////////
/////// PREP & GET & PREPANY
/////////////////////////////////////////////////////////////////////

// runOnce is the heart of the execution logic. It takes a result holder and
// a function to execute, and ensures it runs exactly once, respecting context cancellation.
func runOnce(c *TasksCtx, r *result, execFn func() (any, error)) {
	r.once.Do(func() {
		if err := c.context.Err(); err != nil {
			r.err = err
			return
		}

		type taskOutput struct {
			data any
			err  error
		}
		resultChan := make(chan taskOutput, 1)

		go func() {
			data, err := execFn()
			resultChan <- taskOutput{data: data, err: err}
		}()

		select {
		case <-c.context.Done():
			r.err = c.context.Err()
		case res := <-resultChan:
			r.data = res.data
			r.err = res.err
		}
	})
}

// get contains the memoization logic for type-safe calls.
func get[I, O any](c *TasksCtx, task *RegisteredTask[I, O], input any) (O, error) {
	r := c.getResult(task)

	// Use the shared runOnce helper to perform the execution.
	runOnce(c, r, func() (any, error) {
		return task.execute(c, input)
	})

	if r.err != nil {
		var zero O
		return zero, r.err
	}
	if r.data == nil {
		var zero O
		return zero, r.err
	}
	return r.data.(O), r.err
}

func (task *RegisteredTask[I, O]) Prep(c *TasksCtx, input I) *PreparedTask[I, O] {
	return &PreparedTask[I, O]{c: c, task: task, input: input}
}

func (task *RegisteredTask[I, O]) PrepNoInput(c *TasksCtx) *PreparedTask[I, O] {
	return &PreparedTask[I, O]{c: c, task: task, input: genericsutil.None{}}
}

func (task *RegisteredTask[I, O]) Get(c *TasksCtx, input I) (O, error) {
	return get(c, task, input)
}

func (task *RegisteredTask[I, O]) GetNoInput(c *TasksCtx) (O, error) {
	return get(c, task, genericsutil.None{})
}

type AnyPreparedTask interface {
	getTask() AnyRegisteredTask
	getInput() any
	GetAny() (any, error)
}

type PreparedTask[I any, O any] struct {
	c     *TasksCtx
	task  *RegisteredTask[I, O]
	input any
}

func (pt *PreparedTask[I, O]) getTask() AnyRegisteredTask { return pt.task }
func (pt *PreparedTask[I, O]) getInput() any              { return pt.input }
func (pt *PreparedTask[I, O]) GetAny() (any, error) {
	return pt.Get()
}

func (pt *PreparedTask[I, O]) Get() (O, error) {
	return get(pt.c, pt.task, pt.input)
}

// preparedTaskFromAny is the internal implementation for a task prepared via PrepAny.
type preparedTaskFromAny struct {
	c     *TasksCtx
	task  AnyRegisteredTask
	input any
}

func (p *preparedTaskFromAny) getTask() AnyRegisteredTask { return p.task }
func (p *preparedTaskFromAny) getInput() any              { return p.input }
func (p *preparedTaskFromAny) GetAny() (any, error) {
	r := p.c.getResult(p.task)
	// Use the shared runOnce helper to perform the execution.
	runOnce(p.c, r, func() (any, error) {
		// The `execute` method on the interface handles the erased types.
		return p.task.execute(p.c, p.input)
	})
	return r.data, r.err
}

// PrepAny prepares a type-erased task for execution. It is designed for use
// in generic routers or other dynamic scenarios where the concrete task
// types are not known at compile time.
func PrepAny(c *TasksCtx, task AnyRegisteredTask, input any) AnyPreparedTask {
	return &preparedTaskFromAny{c: c, task: task, input: input}
}

/////////////////////////////////////////////////////////////////////
/////// PARALLEL PRELOAD
/////////////////////////////////////////////////////////////////////

func (c *TasksCtx) ParallelPreload(preparedTasks ...AnyPreparedTask) bool {
	if len(preparedTasks) == 0 {
		return true
	}

	// Optimization: Bypass errgroup for a single task.
	if len(preparedTasks) == 1 {
		_, err := preparedTasks[0].GetAny()
		return err == nil
	}

	g, _ := errgroup.WithContext(c.context)

	for _, pt := range preparedTasks {
		taskToRun := pt // Capture loop variable
		g.Go(func() error {
			_, err := taskToRun.GetAny()
			return err
		})
	}

	return g.Wait() == nil
}
