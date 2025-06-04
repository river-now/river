package tasks

import (
	"context"
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
	return &Task[I, O]{fn: fn}
}

func (t *Task[I, O]) Do(ctx *Context, input any) (any, error) {
	typedInput, ok := input.(I)
	if !ok {
		var zero I
		return nil, fmt.Errorf("invalid input type for task: expected %T, got %T", zero, input)
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
	r := ctx.getOrCreateResult(task)
	r.once.Do(func() {
		if err := ctx.parent.Err(); err != nil {
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
		case <-ctx.parent.Done():
			r.err = ctx.parent.Err()
		case res := <-resultChan:
			r.val = res.val
			r.err = res.err
		}
	})
	if r.err != nil {
		var zero O
		return zero, r.err
	}
	return r.val.(O), nil
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
	g, gCtx := errgroup.WithContext(ctx.parent)
	groupCtx := &Context{
		parent:  gCtx,
		results: ctx.results,
	}
	for _, c := range calls {
		call := c
		g.Go(func() error {
			return call.Run(groupCtx)
		})
	}
	return g.Wait()
}
