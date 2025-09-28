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
	task := NewTask(func(c *Ctx, input int) (int, error) {
		return input * 2, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewCtx(context.Background())
		_, err := runTask(ctx, task, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark parallel execution of independent tasks
func BenchmarkParallelIndependentTasks(b *testing.B) {
	task1 := NewTask(func(c *Ctx, input int) (int, error) {
		return input * 2, nil
	})
	task2 := NewTask(func(c *Ctx, input int) (int, error) {
		return input * 3, nil
	})
	task3 := NewTask(func(c *Ctx, input int) (int, error) {
		return input * 4, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewCtx(context.Background())
		var r1, r2, r3 int // Results are assigned directly
		_ = runTasks(ctx,
			task1.Bind(i, &r1),
			task2.Bind(i, &r2),
			task3.Bind(i, &r3),
		)
	}
}

// Benchmark high contention scenario - many goroutines accessing same task
func BenchmarkHighContention(b *testing.B) {
	// Shared task that will be called by many goroutines
	sharedTask := NewTask(func(c *Ctx, _ struct{}) (string, error) {
		time.Sleep(1 * time.Microsecond) // Simulate minimal work
		return "result", nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewCtx(context.Background())

		var wg sync.WaitGroup
		// 10 goroutines all trying to execute the same task
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := runTask(ctx, sharedTask, struct{}{})
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
	baseTask := NewTask(func(c *Ctx, input int) (int, error) {
		return input * 2, nil
	})

	dependentTask := NewTask(func(c *Ctx, input int) (int, error) {
		// Dependency is called via RunTask
		base, err := runTask(c, baseTask, input)
		if err != nil {
			return 0, err
		}
		return base + 10, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := NewCtx(context.Background())
		_, err := runTask(ctx, dependentTask, i)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark memory allocations for task execution
func BenchmarkAllocations(b *testing.B) {
	task := NewTask(func(c *Ctx, input string) (string, error) {
		return "Hello, " + input, nil
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := NewCtx(context.Background())
		_, err := runTask(ctx, task, "World")
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
				tasks[i] = NewTask(func(c *Ctx, input int) (int, error) {
					return input + taskID, nil
				})
			}

			// Pre-allocate slices for use inside the loop
			boundTasks := make([]BoundTask, numTasks)
			results := make([]int, numTasks)

			b.ResetTimer()
			for i := 0; b.Loop(); i++ {
				ctx := NewCtx(context.Background())

				// Create all boundTasks
				for j := range numTasks {
					boundTasks[j] = tasks[j].Bind(i, &results[j])
				}

				// Execute in parallel
				_ = runTasks(ctx, boundTasks...)
			}
		})
	}
}

// Benchmark context cancellation overhead
func BenchmarkContextCancellation(b *testing.B) {
	// The new library handles cancellation automatically, so the task
	// itself doesn't need a select statement for this test to work.
	task := NewTask(func(c *Ctx, input int) (int, error) {
		time.Sleep(10 * time.Millisecond) // Just simulate work
		return input * 2, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parent, cancel := context.WithCancel(context.Background())
		ctx := NewCtx(parent)

		// Cancel context after a very short time
		time.AfterFunc(1*time.Microsecond, cancel)

		_, _ = runTask(ctx, task, i) // We expect this to be cancelled
	}
}

// Benchmark repeated calls to same task (tests memoization "hot path")
func BenchmarkRepeatedTaskCalls(b *testing.B) {
	var counter int64
	task := NewTask(func(c *Ctx, input int) (int, error) {
		atomic.AddInt64(&counter, 1)
		return input * 2, nil
	})

	// Create a single context and "prime" the cache by running the task once.
	ctx := NewCtx(context.Background())
	_, err := runTask(ctx, task, 42)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// All subsequent calls within this benchmark loop should hit the cache.
		_, err := runTask(ctx, task, 42)
		if err != nil {
			b.Fatal(err)
		}
	}

	// Verify task only ran once across the entire benchmark lifetime
	if atomic.LoadInt64(&counter) != 1 {
		b.Fatalf("Expected task to run once, ran %d times", counter)
	}
}
