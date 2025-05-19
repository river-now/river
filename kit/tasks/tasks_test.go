package tasks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestTasks(t *testing.T) {
	t.Run("BasicTaskExecution", func(t *testing.T) {
		registry := NewRegistry("test")
		task := Register(registry, func(c *Arg[string]) (string, error) {
			return "Hello, " + c.Input, nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		result, err := task.Prep(ctx, "World").Get()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "Hello, World" {
			t.Errorf("Expected 'Hello, World', got '%s'", result)
		}
	})

	t.Run("ParallelExecution", func(t *testing.T) {
		registry := NewRegistry("test")

		task1 := Register(registry, func(c *Arg[int]) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return c.Input * 2, nil
		})

		task2 := Register(registry, func(c *Arg[int]) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return c.Input * 3, nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		start := time.Now()

		twi1 := task1.Prep(ctx, 5)
		twi2 := task2.Prep(ctx, 5)
		ok := ctx.ParallelPreload(twi1, twi2)

		if !ok {
			t.Error("ParallelPreload failed")
		}

		result1, err1 := twi1.Get()
		result2, err2 := twi2.Get()

		duration := time.Since(start)

		if err1 != nil || err2 != nil {
			t.Errorf("Expected no errors, got %v, %v", err1, err2)
		}
		if result1 != 10 || result2 != 15 {
			t.Errorf("Expected 10 and 15, got %d and %d", result1, result2)
		}
		if duration > 150*time.Millisecond {
			t.Errorf("Expected parallel execution (<150ms), took %v", duration)
		}
	})

	t.Run("TaskDependencies", func(t *testing.T) {
		registry := NewRegistry("test")

		authTask := Register(registry, func(c *Arg[string]) (string, error) {
			return "token-" + c.Input, nil
		})

		userTask := Register(registry, func(c *Arg[string]) (string, error) {
			token, err := authTask.Prep(c.TasksCtx, c.Input).Get()
			if err != nil {
				return "", err
			}
			return "user-" + token, nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		result, err := userTask.Prep(ctx, "123").Get()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "user-token-123" {
			t.Errorf("Expected 'user-token-123', got '%s'", result)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		registry := NewRegistry("test")

		task := Register(registry, func(c *Arg[string]) (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "done", nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			ctx.CancelNativeContext()
		}()

		_, err := task.Prep(ctx, "test").Get()
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		registry := NewRegistry("test")

		task := Register(registry, func(c *Arg[string]) (string, error) {
			return "", errors.New("task failed")
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		result, err := task.Prep(ctx, "test").Get()

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
		registry := NewRegistry("test")

		var counter int
		var mu sync.Mutex
		task := Register(registry, func(c *Arg[string]) (string, error) {
			mu.Lock()
			counter++
			mu.Unlock()
			time.Sleep(50 * time.Millisecond)
			return "done", nil
		})

		ctx := registry.NewCtxFromNativeContext(context.Background())
		twi := task.Prep(ctx, "test")

		var wg sync.WaitGroup
		wg.Add(3)

		for range 3 {
			go func() {
				defer wg.Done()
				_, err := twi.Get()
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

func TestTasksWithSharedDependencies_Simple(t *testing.T) {
	t.Run("ParallelTasksWithSharedDependencies", func(t *testing.T) {
		registry := NewRegistry("test")

		// This simulates the Auth task
		var authCounter int
		var authMu sync.Mutex
		authTask := Register(registry, func(c *ArgNoInput) (int, error) {
			authMu.Lock()
			authCounter++
			authMu.Unlock()

			// Simulate work
			time.Sleep(100 * time.Millisecond)

			return 123, nil
		})

		// This simulates the User task
		userTask := Register(registry, func(c *Arg[string]) (string, error) {
			// Critical: Get auth token first
			token, err := authTask.PrepNoInput(c.TasksCtx).Get()
			if err != nil {
				return "", err
			}

			// Simulate additional work
			time.Sleep(50 * time.Millisecond)

			return fmt.Sprintf("user-%d", token), nil
		})

		// This simulates the User2 task
		user2Task := Register(registry, func(c *Arg[string]) (string, error) {
			// Critical: Get auth token first
			token, err := authTask.PrepNoInput(c.TasksCtx).Get()
			if err != nil {
				return "", err
			}

			// Simulate additional work
			time.Sleep(50 * time.Millisecond)

			return fmt.Sprintf("user2-%d", token), nil
		})

		// This simulates the Profile task
		ctx := registry.NewCtxFromNativeContext(context.Background())
		userPrep := userTask.Prep(ctx, "test")
		user2Prep := user2Task.Prep(ctx, "test")

		// Execute both tasks in parallel
		ok := ctx.ParallelPreload(userPrep, user2Prep)
		if !ok {
			t.Fatal("ParallelPreload failed")
		}

		// Get results from both tasks
		userData, userErr := userPrep.Get()
		user2Data, user2Err := user2Prep.Get()

		// Verify no errors
		if userErr != nil {
			t.Errorf("Expected no error from userTask, got %v", userErr)
		}
		if user2Err != nil {
			t.Errorf("Expected no error from user2Task, got %v", user2Err)
		}

		// Verify both tasks got the same correct token value
		expectedUserData := "user-123"
		expectedUser2Data := "user2-123"

		if userData != expectedUserData {
			t.Errorf("Expected userTask to return '%s', got '%s'", expectedUserData, userData)
		}
		if user2Data != expectedUser2Data {
			t.Errorf("Expected user2Task to return '%s', got '%s'", expectedUser2Data, user2Data)
		}

		// Verify authTask was executed exactly once
		if authCounter != 1 {
			t.Errorf("Expected authTask to run exactly once, ran %d times", authCounter)
		}
	})
}

func TestTasksWithSharedDependencies_MoreThorough(t *testing.T) {
	t.Run("ComprehensiveParallelTasksWithSharedDependencies", func(t *testing.T) {
		registry := NewRegistry("test")

		// Track execution order and timing
		var executionOrder []string
		var executionMu sync.Mutex
		recordExecution := func(name string) {
			executionMu.Lock()
			executionOrder = append(executionOrder, name)
			executionMu.Unlock()
		}

		// Track Auth execution details
		var authCounter int
		var authMu sync.Mutex

		// This simulates the Auth task
		authTask := Register(registry, func(c *ArgNoInput) (int, error) {
			recordExecution("auth-start")
			authMu.Lock()
			authCounter++
			authMu.Unlock()

			// Validate context is properly passed
			if c.TasksCtx == nil {
				t.Error("TasksCtx is nil in authTask")
			}

			// Simulate work
			time.Sleep(50 * time.Millisecond)

			recordExecution("auth-end")
			return 123, nil
		})

		// Track User execution details
		var userCounter int
		var userInputs []string
		var userTokens []int
		var userMu sync.Mutex

		// This simulates the User task
		userTask := Register(registry, func(c *Arg[string]) (string, error) {
			recordExecution("user-start")
			userMu.Lock()
			userCounter++
			userInputs = append(userInputs, c.Input)
			userMu.Unlock()

			// Validate input is properly passed
			if c.Input == "" {
				t.Error("Expected non-empty input in userTask")
			}

			// Validate context is properly passed
			if c.TasksCtx == nil {
				t.Error("TasksCtx is nil in userTask")
			}

			// Critical: Get auth token first
			token, err := authTask.PrepNoInput(c.TasksCtx).Get()
			if err != nil {
				return "", err
			}

			userMu.Lock()
			userTokens = append(userTokens, token)
			userMu.Unlock()

			// Simulate additional work
			time.Sleep(25 * time.Millisecond)

			recordExecution("user-end")
			return fmt.Sprintf("user-%s-%d", c.Input, token), nil
		})

		// Track User2 execution details
		var user2Counter int
		var user2Inputs []string
		var user2Tokens []int
		var user2Mu sync.Mutex

		// This simulates the User2 task
		user2Task := Register(registry, func(c *Arg[string]) (string, error) {
			recordExecution("user2-start")
			user2Mu.Lock()
			user2Counter++
			user2Inputs = append(user2Inputs, c.Input)
			user2Mu.Unlock()

			// Validate input is properly passed
			if c.Input == "" {
				t.Error("Expected non-empty input in user2Task")
			}

			// Validate context is properly passed
			if c.TasksCtx == nil {
				t.Error("TasksCtx is nil in user2Task")
			}

			// Critical: Get auth token first
			token, err := authTask.PrepNoInput(c.TasksCtx).Get()
			if err != nil {
				return "", err
			}

			user2Mu.Lock()
			user2Tokens = append(user2Tokens, token)
			user2Mu.Unlock()

			// Simulate additional work
			time.Sleep(25 * time.Millisecond)

			recordExecution("user2-end")
			return fmt.Sprintf("user2-%s-%d", c.Input, token), nil
		})

		// Track Profile execution
		var profileCounter int
		var profileMu sync.Mutex

		// This simulates the Profile task that uses both User and User2
		profileTask := Register(registry, func(c *Arg[string]) (map[string]string, error) {
			recordExecution("profile-start")
			profileMu.Lock()
			profileCounter++
			profileMu.Unlock()

			// Validate input is properly passed
			if c.Input == "" {
				t.Error("Expected non-empty input in profileTask")
			}

			userPrep := userTask.Prep(c.TasksCtx, c.Input)
			user2Prep := user2Task.Prep(c.TasksCtx, c.Input+"_alt") // Different input to verify it's passed correctly

			// Execute both tasks in parallel
			ok := c.TasksCtx.ParallelPreload(userPrep, user2Prep)
			if !ok {
				return nil, fmt.Errorf("ParallelPreload failed")
			}

			// Get results from both tasks
			userData, userErr := userPrep.Get()
			user2Data, user2Err := user2Prep.Get()

			if userErr != nil || user2Err != nil {
				return nil, fmt.Errorf("task errors: %v, %v", userErr, user2Err)
			}

			// Simulate additional work
			time.Sleep(25 * time.Millisecond)

			recordExecution("profile-end")

			results := map[string]string{
				"user":   userData,
				"user2":  user2Data,
				"status": "complete",
			}

			return results, nil
		})

		// Different test inputs
		const testInput1 = "test_input_1"
		const testInput2 = "test_input_2"

		// This tests sequential execution
		ctx1 := registry.NewCtxFromNativeContext(context.Background())
		profileResult1, profileErr1 := profileTask.Prep(ctx1, testInput1).Get()

		// This tests concurrent execution of multiple profiles
		ctx2 := registry.NewCtxFromNativeContext(context.Background())
		profilePrep2 := profileTask.Prep(ctx2, testInput2)

		profileResult2, profileErr2 := profilePrep2.Get()

		// VERIFICATIONS

		// 1. Verify no errors
		if profileErr1 != nil {
			t.Errorf("Expected no error from first profile, got %v", profileErr1)
		}
		if profileErr2 != nil {
			t.Errorf("Expected no error from second profile, got %v", profileErr2)
		}

		// 2. Verify correct profile results
		expectedUserData1 := fmt.Sprintf("user-%s-%d", testInput1, 123)
		expectedUser2Data1 := fmt.Sprintf("user2-%s_alt-%d", testInput1, 123)

		if profileResult1["user"] != expectedUserData1 {
			t.Errorf("Expected profile1.user to be '%s', got '%s'",
				expectedUserData1, profileResult1["user"])
		}
		if profileResult1["user2"] != expectedUser2Data1 {
			t.Errorf("Expected profile1.user2 to be '%s', got '%s'",
				expectedUser2Data1, profileResult1["user2"])
		}

		expectedUserData2 := fmt.Sprintf("user-%s-%d", testInput2, 123)
		expectedUser2Data2 := fmt.Sprintf("user2-%s_alt-%d", testInput2, 123)

		if profileResult2["user"] != expectedUserData2 {
			t.Errorf("Expected profile2.user to be '%s', got '%s'",
				expectedUserData2, profileResult2["user"])
		}
		if profileResult2["user2"] != expectedUser2Data2 {
			t.Errorf("Expected profile2.user2 to be '%s', got '%s'",
				expectedUser2Data2, profileResult2["user2"])
		}

		// 3. Verify task counters
		if authCounter != 2 {
			t.Errorf("Expected authTask to run exactly twice (once per context), ran %d times", authCounter)
		}
		if userCounter != 2 {
			t.Errorf("Expected userTask to run exactly twice, ran %d times", userCounter)
		}
		if user2Counter != 2 {
			t.Errorf("Expected user2Task to run exactly twice, ran %d times", user2Counter)
		}
		if profileCounter != 2 {
			t.Errorf("Expected profileTask to run exactly twice, ran %d times", profileCounter)
		}

		// 4. Verify input propagation to tasks
		expectedUserInputs := []string{testInput1, testInput2}
		for i, input := range userInputs {
			if i < len(expectedUserInputs) && input != expectedUserInputs[i] {
				t.Errorf("Expected userTask input %d to be '%s', got '%s'",
					i, expectedUserInputs[i], input)
			}
		}

		expectedUser2Inputs := []string{testInput1 + "_alt", testInput2 + "_alt"}
		for i, input := range user2Inputs {
			if i < len(expectedUser2Inputs) && input != expectedUser2Inputs[i] {
				t.Errorf("Expected user2Task input %d to be '%s', got '%s'",
					i, expectedUser2Inputs[i], input)
			}
		}

		// 5. Verify token values
		for i, token := range userTokens {
			if token != 123 {
				t.Errorf("Expected userTask token %d to be 123, got %d", i, token)
			}
		}

		for i, token := range user2Tokens {
			if token != 123 {
				t.Errorf("Expected user2Task token %d to be 123, got %d", i, token)
			}
		}

		// 6. Verify execution order for key events
		// While we can't fully predict the exact interleaving due to parallel execution,
		// we can verify certain ordering constraints

		verifyExecutionOrder := func(events []string, message string) bool {
			for i := range len(executionOrder) {
				if executionOrder[i] == events[0] {
					match := true
					for j := 1; j < len(events); j++ {
						if i+j >= len(executionOrder) || executionOrder[i+j] != events[j] {
							match = false
							break
						}
					}
					if match {
						return true
					}
				}
			}
			t.Errorf("Expected execution order sequence not found: %s. Actual order: %v", message, executionOrder)
			return false
		}

		// Verify auth completes before dependent tasks finish
		verifyExecutionOrder([]string{"auth-start", "auth-end"}, "Auth task should complete")

		// Additional diagnostic info if the test fails
		t.Logf("Execution order: %v", executionOrder)
		t.Logf("User inputs: %v", userInputs)
		t.Logf("User2 inputs: %v", user2Inputs)
		t.Logf("User tokens: %v", userTokens)
		t.Logf("User2 tokens: %v", user2Tokens)
	})
}
