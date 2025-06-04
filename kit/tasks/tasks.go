package tasks

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

type None struct{}

type AnyTask interface {
	Do(ctx *Context, input any) (any, error)
}

type Task[I, O any] struct {
	fn func(ctx *Context, input I) (O, error)
}

func NewTask[I, O any](fn func(ctx *Context, input I) (O, error)) *Task[I, O] {
	if fn == nil {
		return nil
	}
	return &Task[I, O]{fn: fn}
}

func (t *Task[I, O]) Do(ctx *Context, input any) (any, error) {
	if t == nil {
		return nil, errors.New("tasks: called Do on a nil Task pointer")
	}
	if t.fn == nil {
		return nil, errors.New("tasks: Task's underlying function is nil")
	}

	typedInput, ok := input.(I)
	if !ok {
		var zeroI I
		var zeroO O
		return zeroO, fmt.Errorf("invalid input type for task: expected %T, got %T", zeroI, input)
	}
	return Do(ctx, t, typedInput)
}

type result struct {
	once sync.Once
	val  any
	err  error
}

type Context struct {
	parent  context.Context
	mu      sync.Mutex
	results map[any]*result
}

func NewContext(parent context.Context) *Context {
	return &Context{
		parent:  parent,
		results: make(map[any]*result),
	}
}

func (c *Context) Parent() context.Context {
	return c.parent
}

func (c *Context) getOrCreateResult(taskPtr any) *result {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r, ok := c.results[taskPtr]; ok {
		return r
	}
	r := &result{}
	c.results[taskPtr] = r
	return r
}

func Do[I, O any](ctx *Context, task *Task[I, O], input I) (O, error) {
	if task == nil || task.fn == nil {
		var zeroO O
		return zeroO, errors.New("tasks: internal Do called with a nil task or task with a nil function")
	}

	r := ctx.getOrCreateResult(task)
	r.once.Do(func() {
		if err := ctx.Parent().Err(); err != nil {
			r.err = err
			return
		}
		type taskOutput struct {
			val O
			err error
		}
		resultChan := make(chan taskOutput, 1)
		go func() {
			val, err := task.fn(ctx, input)
			resultChan <- taskOutput{val: val, err: err}
		}()
		select {
		case <-ctx.Parent().Done():
			r.err = ctx.Parent().Err()
		case res := <-resultChan:
			r.val = res.val
			r.err = res.err
		}
	})

	if r.err != nil {
		var zeroO O
		return zeroO, r.err
	}
	if r.val == nil {
		var zeroO O
		return zeroO, nil
	}
	val, ok := r.val.(O)
	if !ok {
		var zeroO O
		return zeroO, fmt.Errorf("tasks: result type assertion failed, expected %T got %T", zeroO, r.val)
	}
	return val, nil
}

type Callable interface {
	Run(ctx *Context) error
	IsCallable()
}

type BoundCall[O any] struct {
	runner    func(ctx *Context) (O, error)
	resultPtr *O
}

func Bind[I, O any](task *Task[I, O], input I) *BoundCall[O] {
	if task == nil || task.fn == nil {
		return &BoundCall[O]{
			runner: func(ctx *Context) (O, error) {
				var zero O
				return zero, errors.New("tasks: Bind called with a nil or invalid task")
			},
		}
	}
	return &BoundCall[O]{
		runner: func(ctx *Context) (O, error) {
			return Do(ctx, task, input)
		},
	}
}

func (bc *BoundCall[O]) AssignTo(ptr *O) Callable {
	bc.resultPtr = ptr
	return bc
}

func (bc *BoundCall[O]) Run(ctx *Context) error {
	if bc.runner == nil {
		return errors.New("tasks: BoundCall runner is nil")
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

func Go(ctx *Context, calls ...Callable) error {
	if ctx.Parent().Err() != nil {
		return ctx.Parent().Err()
	}
	g, gCtx := errgroup.WithContext(ctx.parent)
	groupCtx := &Context{
		parent:  gCtx,
		results: ctx.results,
	}
	for _, c := range calls {
		if c == nil {
			continue
		}
		call := c
		g.Go(func() error {
			if groupCtx.Parent().Err() != nil {
				return groupCtx.Parent().Err()
			}
			return call.Run(groupCtx)
		})
	}
	return g.Wait()
}
