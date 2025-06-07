package tasks

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Benchmark single task execution
func BenchmarkSingleTask(b *testing.B) {
	task := NewTask(func(c *TasksCtx, input int) (int, error) {
		return input * 2, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewTasksCtx(context.Background())
		_, err := Do(ctx, task, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark parallel execution of independent tasks
func BenchmarkParallelIndependentTasks(b *testing.B) {
	task1 := NewTask(func(c *TasksCtx, input int) (int, error) {
		return input * 2, nil
	})
	task2 := NewTask(func(c *TasksCtx, input int) (int, error) {
		return input * 3, nil
	})
	task3 := NewTask(func(c *TasksCtx, input int) (int, error) {
		return input * 4, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewTasksCtx(context.Background())
		var r1, r2, r3 int // Results are assigned directly
		_ = Go(ctx,
			Bind(task1, i).AssignTo(&r1),
			Bind(task2, i).AssignTo(&r2),
			Bind(task3, i).AssignTo(&r3),
		)
	}
}

// Benchmark high contention scenario - many goroutines accessing same task
func BenchmarkHighContention(b *testing.B) {
	// Shared task that will be called by many goroutines
	sharedTask := NewTask(func(c *TasksCtx, _ struct{}) (string, error) {
		time.Sleep(1 * time.Microsecond) // Simulate minimal work
		return "result", nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewTasksCtx(context.Background())

		var wg sync.WaitGroup
		// 10 goroutines all trying to execute the same task
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := Do(ctx, sharedTask, struct{}{})
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
	baseTask := NewTask(func(c *TasksCtx, input int) (int, error) {
		return input * 2, nil
	})

	dependentTask := NewTask(func(c *TasksCtx, input int) (int, error) {
		// Dependency is called via Do
		base, err := Do(c, baseTask, input)
		if err != nil {
			return 0, err
		}
		return base + 10, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewTasksCtx(context.Background())
		_, err := Do(ctx, dependentTask, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark memory allocations for task execution
func BenchmarkAllocations(b *testing.B) {
	task := NewTask(func(c *TasksCtx, input string) (string, error) {
		return "Hello, " + input, nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := NewTasksCtx(context.Background())
		_, err := Do(ctx, task, "World")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark varying number of parallel tasks
func BenchmarkParallelScaling(b *testing.B) {
	for _, numTasks := range []int{1, 2, 5, 10, 20, 50} {
		b.Run(fmt.Sprintf("tasks-%d", numTasks), func(b *testing.B) {
			// Create tasks
			tasks := make([]*Task[int, int], numTasks)
			for i := 0; i < numTasks; i++ {
				taskID := i
				tasks[i] = NewTask(func(c *TasksCtx, input int) (int, error) {
					return input + taskID, nil
				})
			}

			// Pre-allocate slices for use inside the loop
			callables := make([]Callable, numTasks)
			results := make([]int, numTasks)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := NewTasksCtx(context.Background())

				// Create all callables
				for j := 0; j < numTasks; j++ {
					callables[j] = Bind(tasks[j], i).AssignTo(&results[j])
				}

				// Execute in parallel
				_ = Go(ctx, callables...)
			}
		})
	}
}

// Benchmark context cancellation overhead
func BenchmarkContextCancellation(b *testing.B) {
	// The new library handles cancellation automatically, so the task
	// itself doesn't need a select statement for this test to work.
	task := NewTask(func(c *TasksCtx, input int) (int, error) {
		time.Sleep(10 * time.Millisecond) // Just simulate work
		return input * 2, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parent, cancel := context.WithCancel(context.Background())
		ctx := NewTasksCtx(parent)

		// Cancel context after a very short time
		time.AfterFunc(1*time.Microsecond, cancel)

		_, _ = Do(ctx, task, i) // We expect this to be cancelled
	}
}

// Benchmark repeated calls to same task (tests memoization "hot path")
func BenchmarkRepeatedTaskCalls(b *testing.B) {
	var counter int64
	task := NewTask(func(c *TasksCtx, input int) (int, error) {
		atomic.AddInt64(&counter, 1)
		return input * 2, nil
	})

	// Create a single context and "prime" the cache by running the task once.
	ctx := NewTasksCtx(context.Background())
	_, err := Do(ctx, task, 42)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// All subsequent calls within this benchmark loop should hit the cache.
		_, err := Do(ctx, task, 42)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Verify task only ran once across the entire benchmark lifetime
	if atomic.LoadInt64(&counter) != 1 {
		b.Fatalf("Expected task to run once, ran %d times", counter)
	}
}
