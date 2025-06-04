package tasks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Benchmark single task execution
func BenchmarkSingleTask(b *testing.B) {
	registry := NewRegistry("bench")
	task := Register(registry, func(c *Arg[int]) (int, error) {
		return c.Input * 2, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := registry.NewCtxFromNativeContext(context.Background())
		_, err := task.Get(ctx, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark parallel execution of independent tasks
func BenchmarkParallelIndependentTasks(b *testing.B) {
	registry := NewRegistry("bench")

	task1 := Register(registry, func(c *Arg[int]) (int, error) {
		return c.Input * 2, nil
	})

	task2 := Register(registry, func(c *Arg[int]) (int, error) {
		return c.Input * 3, nil
	})

	task3 := Register(registry, func(c *Arg[int]) (int, error) {
		return c.Input * 4, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := registry.NewCtxFromNativeContext(context.Background())

		prep1 := task1.Prep(ctx, i)
		prep2 := task2.Prep(ctx, i)
		prep3 := task3.Prep(ctx, i)

		ctx.ParallelPreload(prep1, prep2, prep3)

		prep1.Get()
		prep2.Get()
		prep3.Get()
	}
}

// Benchmark high contention scenario - many goroutines accessing same task
func BenchmarkHighContention(b *testing.B) {
	registry := NewRegistry("bench")

	// Shared task that will be called by many goroutines
	sharedTask := Register(registry, func(c *ArgNoInput) (string, error) {
		time.Sleep(1 * time.Microsecond) // Simulate minimal work
		return "result", nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := registry.NewCtxFromNativeContext(context.Background())

		var wg sync.WaitGroup
		// 10 goroutines all trying to execute the same task
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := sharedTask.GetNoInput(ctx)
				if err != nil {
					b.Error(err)
				}
			}()
		}
		wg.Wait()
	}
}

// Benchmark task with dependencies (measures overhead of dependency resolution)
func BenchmarkTaskWithDependencies(b *testing.B) {
	registry := NewRegistry("bench")

	baseTask := Register(registry, func(c *Arg[int]) (int, error) {
		return c.Input * 2, nil
	})

	dependentTask := Register(registry, func(c *Arg[int]) (int, error) {
		base, err := baseTask.Get(c.TasksCtx, c.Input)
		if err != nil {
			return 0, err
		}
		return base + 10, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := registry.NewCtxFromNativeContext(context.Background())
		_, err := dependentTask.Get(ctx, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark memory allocations for task execution
func BenchmarkAllocations(b *testing.B) {
	registry := NewRegistry("bench")

	task := Register(registry, func(c *Arg[string]) (string, error) {
		return "Hello, " + c.Input, nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := registry.NewCtxFromNativeContext(context.Background())
		_, err := task.Get(ctx, "World")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark varying number of parallel tasks
func BenchmarkParallelScaling(b *testing.B) {
	for _, numTasks := range []int{1, 2, 5, 10, 20, 50} {
		b.Run(fmt.Sprintf("tasks-%d", numTasks), func(b *testing.B) {
			registry := NewRegistry("bench")

			// Create tasks
			tasks := make([]*RegisteredTask[int, int], numTasks)
			for i := 0; i < numTasks; i++ {
				taskID := i
				tasks[i] = Register(registry, func(c *Arg[int]) (int, error) {
					return c.Input + taskID, nil
				})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := registry.NewCtxFromNativeContext(context.Background())

				// Prepare all tasks
				preps := make([]AnyPreparedTask, numTasks)
				for j := 0; j < numTasks; j++ {
					preps[j] = tasks[j].Prep(ctx, i)
				}

				// Execute in parallel
				ctx.ParallelPreload(preps...)

				// Get all results
				for _, prep := range preps {
					prep.GetAny()
				}
			}
		})
	}
}

// Benchmark context cancellation overhead
func BenchmarkContextCancellation(b *testing.B) {
	registry := NewRegistry("bench")

	task := Register(registry, func(c *Arg[int]) (int, error) {
		select {
		case <-c.TasksCtx.NativeContext().Done():
			return 0, c.TasksCtx.NativeContext().Err()
		case <-time.After(10 * time.Millisecond):
			return c.Input * 2, nil
		}
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := registry.NewCtxFromNativeContext(context.Background())

		// Cancel context after 5ms
		go func() {
			time.Sleep(5 * time.Millisecond)
			ctx.CancelNativeContext()
		}()

		_, _ = task.Get(ctx, i) // We expect this to be cancelled
	}
}

// Benchmark repeated calls to same task (tests once-only execution)
func BenchmarkRepeatedTaskCalls(b *testing.B) {
	registry := NewRegistry("bench")

	var counter int64
	task := Register(registry, func(c *Arg[int]) (int, error) {
		counter++
		return c.Input * 2, nil
	})

	ctx := registry.NewCtxFromNativeContext(context.Background())
	prep := task.Prep(ctx, 42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := prep.Get()
		if err != nil {
			b.Fatal(err)
		}
	}

	// Verify task only ran once
	if counter != 1 {
		b.Fatalf("Expected task to run once, ran %d times", counter)
	}
}
