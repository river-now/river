package tasks

// A "Task", as used in this package, is simply a function that takes in input,
// returns data (or an error), and runs a maximum of one time per execution
// context (typically, but not necessarily, a web request/response lifecycle),
// even if invoked repeatedly during the lifetime of the execution context.
//
// One cool thing is that Tasks are automatically protected from circular deps
// by Go's 'compile-time "initialization cycle" errors.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/river-now/river/kit/genericsutil"
)

/////////////////////////////////////////////////////////////////////
/////// ARGS
/////////////////////////////////////////////////////////////////////

// sole argument to TaskFunc
type Arg[I any] struct {
	Input I
	*TasksCtx
}

type ArgNoInput = Arg[genericsutil.None]

type anyArg struct {
	input any
	ctx   *TasksCtx
}

/////////////////////////////////////////////////////////////////////
/////// REGISTERED TASKS
/////////////////////////////////////////////////////////////////////

type AnyRegisteredTask interface {
	genericsutil.AnyIOFunc
	getID() int
}

type ioFunc[I any, O any] = genericsutil.IOFunc[*Arg[I], O]

// returned from tasks.Register(Registry, IOTask)
type RegisteredTask[I any, O any] struct {
	ioFunc[I, O]
	id int
}

func (task RegisteredTask[I, O]) getID() int { return task.id }

// Adds a task to the registry
func Register[I any, O any](tr *Registry, f genericsutil.IOFunc[*Arg[I], O]) *RegisteredTask[I, O] {
	id := tr.count
	tr.count++

	// This will ultimately be called (exactly once) by the TasksCtx.doOnce method
	inner := func(anyArg *anyArg) (O, error) {
		return f(&Arg[I]{
			TasksCtx: anyArg.ctx,
			Input:    genericsutil.AssertOrZero[I](anyArg.input),
		})
	}

	// cast as a typed IO func (adds genericsutil helper methods)
	asIOFunc := genericsutil.IOFunc[*anyArg, O](inner)

	// type erasure
	asAnyIOFunc := genericsutil.AnyIOFunc(asIOFunc)

	// add to registry
	tr.registry[id] = asAnyIOFunc

	// This is the function that will be called by the TasksCtx.doOnce method
	return &RegisteredTask[I, O]{
		id: id,
		ioFunc: func(c *Arg[I]) (O, error) {
			c.TasksCtx.doOnce(id, c.TasksCtx, c.Input)
			c.TasksCtx.mu.Lock()
			defer c.TasksCtx.mu.Unlock()
			result, ok := c.TasksCtx.results.results[id]
			if !ok {
				var o O
				return o, fmt.Errorf("task result not found for task with id: %d", id)
			}
			return result.Data.(O), result.Err
		},
	}
}

/////////////////////////////////////////////////////////////////////
/////// TASKS REGISTRY
/////////////////////////////////////////////////////////////////////

type Registry struct {
	count    int
	registry map[int]genericsutil.AnyIOFunc
}

func (tr *Registry) NewCtxFromNativeContext(parentContext context.Context) *TasksCtx {
	return newTasksCtx(tr, parentContext, nil)
}

func (tr *Registry) NewCtxFromRequest(r *http.Request) *TasksCtx {
	return newTasksCtx(tr, r.Context(), r)
}

func NewRegistry() *Registry {
	return &Registry{registry: make(map[int]genericsutil.AnyIOFunc)}
}

/////////////////////////////////////////////////////////////////////
/////// CTX
/////////////////////////////////////////////////////////////////////

type TasksCtx struct {
	mu       *sync.Mutex
	request  *http.Request
	registry *Registry
	results  *TaskResults

	context context.Context
	cancel  context.CancelFunc
}

func newTasksCtx(registry *Registry, parentContext context.Context, r *http.Request) *TasksCtx {
	contextWithCancel, cancel := context.WithCancel(parentContext)

	c := &TasksCtx{
		mu:       &sync.Mutex{},
		request:  r,
		registry: registry,
		context:  contextWithCancel,
		cancel:   cancel,
	}

	c.results = newResults(c)

	return c
}

func (c *TasksCtx) Request() *http.Request {
	return c.request
}

func (c *TasksCtx) NativeContext() context.Context {
	return c.context
}

func (c *TasksCtx) CancelNativeContext() {
	c.cancel()
}

func (task *RegisteredTask[I, O]) Prep(c *TasksCtx, input I) *PreparedTask[I, O] {
	return &PreparedTask[I, O]{c: c, task: task, input: input}
}

func (task *RegisteredTask[I, O]) PrepNoInput(c *TasksCtx) *PreparedTask[I, O] {
	return &PreparedTask[I, O]{c: c, task: task, input: genericsutil.None{}}
}

func (task *RegisteredTask[I, O]) Get(c *TasksCtx, input I) (O, error) {
	pt := PreparedTask[I, O]{c: c, task: task, input: input}
	return pt.Get()
}

func (task *RegisteredTask[I, O]) GetNoInput(c *TasksCtx) (O, error) {
	pt := PreparedTask[I, O]{c: c, task: task, input: genericsutil.None{}}
	return pt.Get()
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

func (twi *PreparedTask[I, O]) getTask() AnyRegisteredTask { return twi.task }
func (twi *PreparedTask[I, O]) getInput() any              { return twi.input }
func (twi *PreparedTask[I, O]) GetAny() (any, error) {
	return twi.Get()
}

func (twi *PreparedTask[I, O]) Get() (O, error) {
	return twi.task.ioFunc(&Arg[I]{
		TasksCtx: twi.c,
		Input:    genericsutil.AssertOrZero[I](twi.input),
	})
}

type anyPreparedTaskImpl struct {
	c     *TasksCtx
	task  AnyRegisteredTask
	input any
}

func (twi *anyPreparedTaskImpl) getTask() AnyRegisteredTask { return twi.task }
func (twi *anyPreparedTaskImpl) getInput() any              { return twi.input }

func (twi anyPreparedTaskImpl) GetAny() (any, error) {
	twi.c.ParallelPreload(PrepAny(twi.c, twi.task, twi.input))
	x := twi.c.results.results[twi.task.getID()]
	return x.Data, x.Err
}

func PrepAny[I any](c *TasksCtx, task AnyRegisteredTask, input I) AnyPreparedTask {
	return &anyPreparedTaskImpl{c: c, task: task, input: input}
}

func (c *TasksCtx) ParallelPreload(preparedTasks ...AnyPreparedTask) bool {
	if len(preparedTasks) == 0 {
		return true
	}

	if len(preparedTasks) == 1 {
		t := preparedTasks[0]
		c.doOnce(t.getTask().getID(), c, t.getInput())
		return c.results.AllOK()
	}

	var wg sync.WaitGroup
	wg.Add(len(preparedTasks))
	for _, t := range preparedTasks {
		go func() {
			c.doOnce(t.getTask().getID(), c, t.getInput())
			wg.Done()
		}()
	}
	wg.Wait()

	return c.results.AllOK()
}

func (c *TasksCtx) doOnce(taskID int, ctx *TasksCtx, input any) {
	taskHelper := c.registry.registry[taskID]

	c.mu.Lock()
	if _, ok := c.results.results[taskID]; !ok {
		c.results.results[taskID] = &TaskResult{once: &sync.Once{}}
	}
	c.mu.Unlock()

	if c.context.Err() != nil {
		c.mu.Lock()
		c.results.results[taskID].Data = taskHelper.O()
		c.results.results[taskID].Err = errors.New("parent context canceled")
		c.mu.Unlock()
		return
	}

	c.getSyncOnce(taskID).Do(func() {
		// check if context is canceled
		if c.context.Err() != nil {
			c.mu.Lock()
			c.results.results[taskID].Data = taskHelper.O()
			c.results.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
			return
		}

		resultChan := make(chan *TaskResult, 1)
		go func() {
			data, err := taskHelper.ExecuteStrict(&anyArg{input: input, ctx: ctx})
			resultChan <- &TaskResult{Data: data, Err: err}
		}()

		select {
		case <-c.context.Done():
			c.mu.Lock()
			c.results.results[taskID].Data = taskHelper.O()
			c.results.results[taskID].Err = c.context.Err()
			c.mu.Unlock()
		case result := <-resultChan:
			c.mu.Lock()
			c.results.results[taskID].Data = result.Data
			c.results.results[taskID].Err = result.Err
			c.mu.Unlock()
		}
	})
}

func (c *TasksCtx) getSyncOnce(taskID int) *sync.Once {
	c.mu.Lock()
	defer c.mu.Unlock()
	result, ok := c.results.results[taskID]
	if !ok {
		result = newTaskResult()
		c.results.results[taskID] = result
	}
	return result.once
}

/////////////////////////////////////////////////////////////////////
/////// RESULTS
/////////////////////////////////////////////////////////////////////

type TaskResult struct {
	Data any
	Err  error
	once *sync.Once
}

func newTaskResult() *TaskResult {
	return &TaskResult{once: &sync.Once{}}
}

func (r *TaskResult) OK() bool {
	return r.Err == nil
}

type TaskResults struct {
	c       *TasksCtx
	results map[int]*TaskResult
}

func newResults(c *TasksCtx) *TaskResults {
	return &TaskResults{
		c:       c,
		results: make(map[int]*TaskResult),
	}
}

func (tr TaskResults) AllOK() bool {
	for _, result := range tr.results {
		if !result.OK() {
			return false
		}
	}
	return true
}
