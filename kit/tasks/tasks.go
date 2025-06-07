package tasks

// A "Task", as used in this package, is simply a function that takes in input,
// returns data (or an error), and runs a maximum of one time per execution
// context / input value pairing (typically, but not necessarily, a web request
// lifecycle), even if invoked repeatedly during the lifetime of the execution
// context.
//
// One cool thing is that Tasks are automatically protected from circular deps
// by Go's compile-time "initialization cycle" errors (assuming they are defined
// via top-level var declarations).

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/river-now/river/kit/genericsutil"
	"golang.org/x/sync/errgroup"
)

type AnyTask interface {
	Do(ctx *TasksCtx, input any) (any, error)
}

type Task[I comparable, O any] struct {
	fn func(ctx *TasksCtx, input I) (O, error)
}

func NewTask[I comparable, O any](fn func(ctx *TasksCtx, input I) (O, error)) *Task[I, O] {
	if fn == nil {
		return nil
	}
	return &Task[I, O]{fn: fn}
}

func (t *Task[I, O]) Do(ctx *TasksCtx, input any) (any, error) {
	if t == nil {
		return nil, errors.New("tasks: called Do on a nil Task pointer")
	}
	if t.fn == nil {
		return nil, errors.New("tasks: Task's underlying function is nil")
	}
	return Do(ctx, t, genericsutil.AssertOrZero[I](input))
}

// taskKey is used for map lookups to avoid allocating anonymous structs
type taskKey struct {
	taskPtr uintptr
	input   any
}

type TasksCtx struct {
	mu      *sync.RWMutex
	results map[taskKey]*TaskResult
	ctx     context.Context
}

func NewTasksCtx(parent context.Context) *TasksCtx {
	if parent == nil {
		parent = context.Background()
	}
	return &TasksCtx{
		mu:      &sync.RWMutex{},
		results: make(map[taskKey]*TaskResult, 4), // Pre-allocate for typical request size
		ctx:     parent,
	}
}

func (c *TasksCtx) NativeContext() context.Context {
	return c.ctx
}

func Do[I comparable, O any](c *TasksCtx, task *Task[I, O], input I) (result O, err error) {
	if c == nil {
		return result, errors.New("tasks: nil TasksCtx")
	}
	if task == nil || task.fn == nil {
		return result, errors.New("tasks: invalid task")
	}

	// Check context only once at the beginning
	if err := c.ctx.Err(); err != nil {
		return result, err
	}

	r := c.getOrCreateResult(task, input)
	r.once.Do(func() {
		val, err := task.fn(c, input)
		if err != nil {
			r.Err = err
			return
		}
		if cerr := c.ctx.Err(); cerr != nil {
			r.Err = cerr
			return
		}
		r.Data = val
		r.Err = nil
	})

	if r.Err != nil {
		return result, r.Err
	}
	if r.Data == nil {
		return result, nil
	}
	return genericsutil.AssertOrZero[O](r.Data), nil
}

func (c *TasksCtx) getOrCreateResult(taskPtr any, input any) *TaskResult {
	// Use uintptr for task pointer to avoid allocation
	key := taskKey{
		taskPtr: reflect.ValueOf(taskPtr).Pointer(),
		input:   input,
	}

	c.mu.RLock()
	if r, ok := c.results[key]; ok {
		c.mu.RUnlock()
		return r
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if r, ok := c.results[key]; ok {
		return r
	}

	r := newTaskResult()
	c.results[key] = r
	return r
}

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

type Callable interface {
	Run(ctx *TasksCtx) error
	IsCallable()
}

type BoundCall[O any] struct {
	runner    func(ctx *TasksCtx) (O, error)
	resultPtr *O
}

func Bind[I comparable, O any](task *Task[I, O], input I) *BoundCall[O] {
	if task == nil || task.fn == nil {
		return &BoundCall[O]{
			runner: func(ctx *TasksCtx) (O, error) {
				var zero O
				return zero, errors.New("tasks: Bind called with a nil or invalid task")
			},
		}
	}
	return &BoundCall[O]{
		runner: func(ctx *TasksCtx) (O, error) {
			return Do(ctx, task, input)
		},
	}
}

func (bc *BoundCall[O]) AssignTo(ptr *O) Callable {
	bc.resultPtr = ptr
	return bc
}

func (bc *BoundCall[O]) Run(ctx *TasksCtx) error {
	if ctx == nil {
		return errors.New("tasks: BoundCall.Run called with nil TasksCtx")
	}
	if bc.runner == nil {
		return errors.New("tasks: BoundCall runner is nil (task may have been invalid at Bind)")
	}
	res, err := bc.runner(ctx)
	if err != nil {
		return err
	}
	if bc.resultPtr != nil {
		*bc.resultPtr = res
	}
	return nil
}

func (bc *BoundCall[O]) IsCallable() {}

func Go(ctx *TasksCtx, calls ...Callable) error {
	if ctx == nil {
		return errors.New("tasks: Go called with nil TasksCtx")
	}
	if err := ctx.ctx.Err(); err != nil {
		return err
	}
	valid := calls[:0]
	for _, c := range calls {
		if c != nil {
			valid = append(valid, c)
		}
	}
	switch len(valid) {
	case 0:
		return nil
	case 1:
		return valid[0].Run(ctx)
	}
	g, gCtx := errgroup.WithContext(ctx.ctx)
	shared := &TasksCtx{
		mu:      ctx.mu,
		results: ctx.results,
		ctx:     gCtx,
	}
	for _, call := range valid {
		c := call
		g.Go(func() error {
			if err := c.Run(shared); err != nil {
				return err
			}
			return shared.ctx.Err()
		})
	}
	return g.Wait()
}
