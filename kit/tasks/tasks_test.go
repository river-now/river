package tasks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTasks(t *testing.T) {
	t.Run("BasicTaskExecution", func(t *testing.T) {
		task := NewTask(func(c *Ctx, input string) (string, error) {
			return "Hello, " + input, nil
		})

		ctx := NewCtx(context.Background())
		result, err := task.Run(ctx, "World")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "Hello, World" {
			t.Errorf("Expected 'Hello, World', got '%s'", result)
		}
	})

	t.Run("ParallelExecution", func(t *testing.T) {
		task1 := NewTask(func(c *Ctx, input int) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return input * 2, nil
		})

		task2 := NewTask(func(c *Ctx, input string) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return input + "3", nil
		})

		ctx := NewCtx(context.Background())
		start := time.Now()

		var result1 int
		var result2 string
		err := ctx.RunParallel(
			task1.Bind(5, &result1),
			task2.Bind("3", &result2),
		)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no errors, got %v", err)
		}
		if result1 != 10 || result2 != "33" {
			t.Errorf("Expected 10 and 15, got %d and %s", result1, result2)
		}
		if duration > 150*time.Millisecond {
			t.Errorf("Expected parallel execution (<150ms), took %v", duration)
		}
	})

	t.Run("TaskDependencies", func(t *testing.T) {
		authTask := NewTask(func(c *Ctx, input string) (string, error) {
			return "token-" + input, nil
		})

		userTask := NewTask(func(c *Ctx, input string) (string, error) {
			token, err := authTask.Run(c, input)
			if err != nil {
				return "", err
			}
			return "user-" + token, nil
		})

		ctx := NewCtx(context.Background())
		result, err := userTask.Run(ctx, "123")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "user-token-123" {
			t.Errorf("Expected 'user-token-123', got '%s'", result)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		task := NewTask(func(c *Ctx, _ string) (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "done", nil
		})

		parentCtx, cancel := context.WithCancel(context.Background())
		ctx := NewCtx(parentCtx)

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := runTask(ctx, task, "test")
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		task := NewTask(func(c *Ctx, _ string) (string, error) {
			return "", errors.New("task failed")
		})

		ctx := NewCtx(context.Background())
		result, err := runTask(ctx, task, "test")

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if err.Error() != "task failed" {
			t.Errorf("Expected 'task failed' error, got '%v'", err)
		}
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("OnceExecution", func(t *testing.T) {
		var counter int32
		task := NewTask(func(c *Ctx, _ string) (string, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(50 * time.Millisecond)
			return "done", nil
		})

		ctx := NewCtx(context.Background())
		var wg sync.WaitGroup
		wg.Add(3)

		for range 3 {
			go func() {
				defer wg.Done()
				_, err := runTask(ctx, task, "test")
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}()
		}
		wg.Wait()

		if counter != 1 {
			t.Errorf("Expected task to run once, ran %d times", counter)
		}
	})
}

func TestTasksWithSharedDependencies(t *testing.T) {
	t.Run("ParallelTasksWithSharedDependencies", func(t *testing.T) {
		var authCounter int32
		authTask := NewTask(func(c *Ctx, _ struct{}) (int, error) {
			atomic.AddInt32(&authCounter, 1)
			time.Sleep(100 * time.Millisecond)
			return 123, nil
		})

		userTask := NewTask(func(c *Ctx, _ string) (string, error) {
			token, err := runTask(c, authTask, struct{}{})
			if err != nil {
				return "", err
			}
			time.Sleep(50 * time.Millisecond)
			return fmt.Sprintf("user-%d", token), nil
		})

		user2Task := NewTask(func(c *Ctx, _ string) (string, error) {
			token, err := runTask(c, authTask, struct{}{})
			if err != nil {
				return "", err
			}
			time.Sleep(50 * time.Millisecond)
			return fmt.Sprintf("user2-%d", token), nil
		})

		ctx := NewCtx(context.Background())
		var userData, user2Data string
		err := ctx.RunParallel(
			userTask.Bind("test", &userData),
			user2Task.Bind("test", &user2Data),
		)

		if err != nil {
			t.Fatal("runTasks failed with an unexpected error:", err)
		}

		expectedUserData := "user-123"
		expectedUser2Data := "user2-123"
		if userData != expectedUserData {
			t.Errorf("Expected userTask to return '%s', got '%s'", expectedUserData, userData)
		}
		if user2Data != expectedUser2Data {
			t.Errorf("Expected user2Task to return '%s', got '%s'", expectedUser2Data, user2Data)
		}

		if authCounter != 1 {
			t.Errorf("Expected authTask to run exactly once, ran %d times", authCounter)
		}
	})
}

func TestComprehensiveSharedDependencies(t *testing.T) {
	// State tracking variables for the test
	var executionOrder []string
	var executionMu sync.Mutex
	recordExecution := func(name string) {
		executionMu.Lock()
		executionOrder = append(executionOrder, name)
		executionMu.Unlock()
	}

	var authCounter, userCounter, user2Counter, profileCounter int32
	var userInputs, user2Inputs []string
	var userTokens, user2Tokens []int
	var stateMu sync.Mutex // A single mutex to protect all test state slices/maps

	// Define Tasks using the new API
	authTask := NewTask(func(c *Ctx, _ struct{}) (int, error) {
		recordExecution("auth-start")
		atomic.AddInt32(&authCounter, 1)
		time.Sleep(50 * time.Millisecond)
		recordExecution("auth-end")
		return 123, nil
	})

	userTask := NewTask(func(c *Ctx, input string) (string, error) {
		recordExecution("user-start")
		atomic.AddInt32(&userCounter, 1)
		if input == "" {
			t.Error("Expected non-empty input in userTask")
		}

		token, err := runTask(c, authTask, struct{}{})
		if err != nil {
			return "", err
		}

		stateMu.Lock()
		userInputs = append(userInputs, input)
		userTokens = append(userTokens, token)
		stateMu.Unlock()

		time.Sleep(25 * time.Millisecond)
		recordExecution("user-end")
		return fmt.Sprintf("user-%s-%d", input, token), nil
	})

	user2Task := NewTask(func(c *Ctx, input string) (string, error) {
		recordExecution("user2-start")
		atomic.AddInt32(&user2Counter, 1)
		if input == "" {
			t.Error("Expected non-empty input in user2Task")
		}

		token, err := runTask(c, authTask, struct{}{})
		if err != nil {
			return "", err
		}

		stateMu.Lock()
		user2Inputs = append(user2Inputs, input)
		user2Tokens = append(user2Tokens, token)
		stateMu.Unlock()

		time.Sleep(25 * time.Millisecond)
		recordExecution("user2-end")
		return fmt.Sprintf("user2-%s-%d", input, token), nil
	})

	profileTask := NewTask(func(ctx *Ctx, input string) (map[string]string, error) {
		recordExecution("profile-start")
		atomic.AddInt32(&profileCounter, 1)

		var userData, user2Data string
		err := ctx.RunParallel(
			userTask.Bind(input, &userData),
			user2Task.Bind(input+"_alt", &user2Data),
		)
		if err != nil {
			return nil, err
		}

		time.Sleep(25 * time.Millisecond)
		recordExecution("profile-end")

		return map[string]string{
			"user":   userData,
			"user2":  user2Data,
			"status": "complete",
		}, nil
	})

	// --- Execute Test Cases ---
	const testInput1 = "test_input_1"
	const testInput2 = "test_input_2"

	// Execution for first context
	ctx1 := NewCtx(context.Background())
	profileResult1, profileErr1 := runTask(ctx1, profileTask, testInput1)

	// Execution for second context
	ctx2 := NewCtx(context.Background())
	profileResult2, profileErr2 := runTask(ctx2, profileTask, testInput2)

	// --- VERIFICATIONS ---
	if profileErr1 != nil {
		t.Errorf("Expected no error from first profile, got %v", profileErr1)
	}
	if profileErr2 != nil {
		t.Errorf("Expected no error from second profile, got %v", profileErr2)
	}

	expectedUserData1 := "user-test_input_1-123"
	expectedUser2Data1 := "user2-test_input_1_alt-123"
	if profileResult1["user"] != expectedUserData1 {
		t.Errorf("Expected profile1.user to be '%s', got '%s'", expectedUserData1, profileResult1["user"])
	}
	if profileResult1["user2"] != expectedUser2Data1 {
		t.Errorf("Expected profile1.user2 to be '%s', got '%s'", expectedUser2Data1, profileResult1["user2"])
	}

	expectedUserData2 := "user-test_input_2-123"
	expectedUser2Data2 := "user2-test_input_2_alt-123"
	if profileResult2["user"] != expectedUserData2 {
		t.Errorf("Expected profile2.user to be '%s', got '%s'", expectedUserData2, profileResult2["user"])
	}
	if profileResult2["user2"] != expectedUser2Data2 {
		t.Errorf("Expected profile2.user2 to be '%s', got '%s'", expectedUser2Data2, profileResult2["user2"])
	}

	if atomic.LoadInt32(&authCounter) != 2 {
		t.Errorf("Expected authTask to run twice (once per context), ran %d times", authCounter)
	}
	if atomic.LoadInt32(&userCounter) != 2 {
		t.Errorf("Expected userTask to run twice, ran %d times", userCounter)
	}
	if atomic.LoadInt32(&user2Counter) != 2 {
		t.Errorf("Expected user2Task to run twice, ran %d times", user2Counter)
	}
	if atomic.LoadInt32(&profileCounter) != 2 {
		t.Errorf("Expected profileTask to run twice, ran %d times", profileCounter)
	}

	// Verify execution order for key events
	verifyExecutionOrder := func(events [2]string, message string) {
		executionMu.Lock()
		defer executionMu.Unlock()
		firstIndex := -1
		// Find the first event
		for i, event := range executionOrder {
			if event == events[0] {
				firstIndex = i
				break
			}
		}
		if firstIndex == -1 {
			t.Errorf("Execution order event not found: %s. Order: %v", events[0], executionOrder)
			return
		}
		// Search for the second event *after* the first one
		for i := firstIndex + 1; i < len(executionOrder); i++ {
			if executionOrder[i] == events[1] {
				return // Found in correct order
			}
		}
		t.Errorf("Expected '%s' to appear after '%s', but it didn't. Order: %v. Message: %s", events[1], events[0], executionOrder, message)
	}

	// Check that auth completes before its dependents can finish
	verifyExecutionOrder([2]string{"auth-end", "user-end"}, "userTask should finish after authTask")
	verifyExecutionOrder([2]string{"auth-end", "user2-end"}, "user2Task should finish after authTask")

	// Check that the top-level profile task finishes after its own dependents
	verifyExecutionOrder([2]string{"user-end", "profile-end"}, "profileTask should finish after userTask")
	verifyExecutionOrder([2]string{"user2-end", "profile-end"}, "profileTask should finish after user2Task")

	// Log for diagnostics if needed
	t.Logf("Execution order: %v", executionOrder)
}

func TestTasksWithDifferentInputs(t *testing.T) {
	t.Run("Same_Input_Uses_Cache", func(t *testing.T) {
		var execCount int32
		task := NewTask(func(ctx *Ctx, input string) (string, error) {
			atomic.AddInt32(&execCount, 1)
			return "result-" + input, nil
		})

		ctx := NewCtx(context.Background())

		// Call 3 times with same input
		r1, _ := runTask(ctx, task, "foo")
		r2, _ := runTask(ctx, task, "foo")
		r3, _ := runTask(ctx, task, "foo")

		if r1 != "result-foo" || r2 != "result-foo" || r3 != "result-foo" {
			t.Error("Expected same result for same input")
		}

		if execCount != 1 {
			t.Errorf("Expected task to execute once, executed %d times", execCount)
		}
	})

	t.Run("Different_Inputs_Execute_Separately", func(t *testing.T) {
		var execCount int32
		execInputs := make([]string, 0)
		var mu sync.Mutex

		task := NewTask(func(ctx *Ctx, input string) (string, error) {
			atomic.AddInt32(&execCount, 1)
			mu.Lock()
			execInputs = append(execInputs, input)
			mu.Unlock()
			return "result-" + input, nil
		})

		ctx := NewCtx(context.Background())

		// Call with different inputs
		r1, _ := runTask(ctx, task, "foo")
		r2, _ := runTask(ctx, task, "bar")
		r3, _ := runTask(ctx, task, "baz")

		// Call again with same inputs (should use cache)
		r1b, _ := runTask(ctx, task, "foo")
		r2b, _ := runTask(ctx, task, "bar")

		// Verify results
		if r1 != "result-foo" || r1b != "result-foo" {
			t.Error("Expected consistent results for 'foo'")
		}
		if r2 != "result-bar" || r2b != "result-bar" {
			t.Error("Expected consistent results for 'bar'")
		}
		if r3 != "result-baz" {
			t.Error("Expected correct result for 'baz'")
		}

		// Should execute exactly 3 times (once per unique input)
		if execCount != 3 {
			t.Errorf("Expected 3 executions, got %d", execCount)
		}

		// Verify it saw all 3 inputs
		if len(execInputs) != 3 {
			t.Errorf("Expected 3 inputs recorded, got %d", len(execInputs))
		}
	})

	t.Run("Different_Input_Types", func(t *testing.T) {
		// Test with int inputs
		intTask := NewTask(func(ctx *Ctx, input int) (int, error) {
			return input * 2, nil
		})

		ctx := NewCtx(context.Background())

		r1, _ := runTask(ctx, intTask, 5)
		r2, _ := runTask(ctx, intTask, 10)
		r3, _ := runTask(ctx, intTask, 5) // Same as r1

		if r1 != 10 || r3 != 10 {
			t.Error("Expected same result for same int input")
		}
		if r2 != 20 {
			t.Error("Expected different result for different int input")
		}
	})

	t.Run("Struct_Inputs", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}

		var execCount int32
		task := NewTask(func(ctx *Ctx, p Person) (string, error) {
			atomic.AddInt32(&execCount, 1)
			return fmt.Sprintf("%s is %d", p.Name, p.Age), nil
		})

		ctx := NewCtx(context.Background())

		p1 := Person{Name: "Alice", Age: 30}
		p2 := Person{Name: "Bob", Age: 25}

		r1, _ := runTask(ctx, task, p1)
		r2, _ := runTask(ctx, task, p2)
		r3, _ := runTask(ctx, task, p1) // Same as first

		if r1 != "Alice is 30" || r3 != "Alice is 30" {
			t.Error("Expected same result for same struct")
		}
		if r2 != "Bob is 25" {
			t.Error("Expected different result for different struct")
		}

		if execCount != 2 {
			t.Errorf("Expected 2 executions, got %d", execCount)
		}
	})

	t.Run("Parallel_Different_Inputs", func(t *testing.T) {
		var execCount int32
		task := NewTask(func(ctx *Ctx, input string) (string, error) {
			atomic.AddInt32(&execCount, 1)
			time.Sleep(50 * time.Millisecond)
			return "result-" + input, nil
		})

		ctx := NewCtx(context.Background())

		var result1, result2, result3 string
		err := ctx.RunParallel(
			task.Bind("alpha", &result1),
			task.Bind("beta", &result2),
			task.Bind("alpha", &result3), // Duplicate
		)

		if err != nil {
			t.Fatal(err)
		}

		if result1 != "result-alpha" || result3 != "result-alpha" {
			t.Error("Expected same result for 'alpha'")
		}
		if result2 != "result-beta" {
			t.Error("Expected different result for 'beta'")
		}

		// Should only execute twice (alpha and beta)
		if execCount != 2 {
			t.Errorf("Expected 2 executions, got %d", execCount)
		}
	})
}
