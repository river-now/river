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
		task := NewTask(func(c *Context, input string) (string, error) {
			return "Hello, " + input, nil
		})

		ctx := NewContext(context.Background())
		result, err := Do(ctx, task, "World")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "Hello, World" {
			t.Errorf("Expected 'Hello, World', got '%s'", result)
		}
	})

	t.Run("ParallelExecution", func(t *testing.T) {
		task1 := NewTask(func(c *Context, input int) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return input * 2, nil
		})

		task2 := NewTask(func(c *Context, input int) (int, error) {
			time.Sleep(100 * time.Millisecond)
			return input * 3, nil
		})

		ctx := NewContext(context.Background())
		start := time.Now()

		var result1, result2 int
		err := Go(ctx,
			Bind(task1, 5).AssignTo(&result1),
			Bind(task2, 5).AssignTo(&result2),
		)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Expected no errors, got %v", err)
		}
		if result1 != 10 || result2 != 15 {
			t.Errorf("Expected 10 and 15, got %d and %d", result1, result2)
		}
		if duration > 150*time.Millisecond {
			t.Errorf("Expected parallel execution (<150ms), took %v", duration)
		}
	})

	t.Run("TaskDependencies", func(t *testing.T) {
		authTask := NewTask(func(c *Context, input string) (string, error) {
			return "token-" + input, nil
		})

		userTask := NewTask(func(c *Context, input string) (string, error) {
			token, err := Do(c, authTask, input)
			if err != nil {
				return "", err
			}
			return "user-" + token, nil
		})

		ctx := NewContext(context.Background())
		result, err := Do(ctx, userTask, "123")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "user-token-123" {
			t.Errorf("Expected 'user-token-123', got '%s'", result)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		task := NewTask(func(c *Context, _ string) (string, error) {
			time.Sleep(200 * time.Millisecond)
			return "done", nil
		})

		parentCtx, cancel := context.WithCancel(context.Background())
		ctx := NewContext(parentCtx)

		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := Do(ctx, task, "test")
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		task := NewTask(func(c *Context, _ string) (string, error) {
			return "", errors.New("task failed")
		})

		ctx := NewContext(context.Background())
		result, err := Do(ctx, task, "test")

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
		task := NewTask(func(c *Context, _ string) (string, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(50 * time.Millisecond)
			return "done", nil
		})

		ctx := NewContext(context.Background())
		var wg sync.WaitGroup
		wg.Add(3)

		for range 3 {
			go func() {
				defer wg.Done()
				_, err := Do(ctx, task, "test")
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
		authTask := NewTask(func(c *Context, _ None) (int, error) {
			atomic.AddInt32(&authCounter, 1)
			time.Sleep(100 * time.Millisecond)
			return 123, nil
		})

		userTask := NewTask(func(c *Context, _ string) (string, error) {
			token, err := Do(c, authTask, None{})
			if err != nil {
				return "", err
			}
			time.Sleep(50 * time.Millisecond)
			return fmt.Sprintf("user-%d", token), nil
		})

		user2Task := NewTask(func(c *Context, _ string) (string, error) {
			token, err := Do(c, authTask, None{})
			if err != nil {
				return "", err
			}
			time.Sleep(50 * time.Millisecond)
			return fmt.Sprintf("user2-%d", token), nil
		})

		ctx := NewContext(context.Background())
		var userData, user2Data string
		err := Go(ctx,
			Bind(userTask, "test").AssignTo(&userData),
			Bind(user2Task, "test").AssignTo(&user2Data),
		)

		if err != nil {
			t.Fatal("Go failed with an unexpected error:", err)
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
	authTask := NewTask(func(c *Context, _ None) (int, error) {
		recordExecution("auth-start")
		atomic.AddInt32(&authCounter, 1)
		time.Sleep(50 * time.Millisecond)
		recordExecution("auth-end")
		return 123, nil
	})

	userTask := NewTask(func(c *Context, input string) (string, error) {
		recordExecution("user-start")
		atomic.AddInt32(&userCounter, 1)
		if input == "" {
			t.Error("Expected non-empty input in userTask")
		}

		token, err := Do(c, authTask, None{})
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

	user2Task := NewTask(func(c *Context, input string) (string, error) {
		recordExecution("user2-start")
		atomic.AddInt32(&user2Counter, 1)
		if input == "" {
			t.Error("Expected non-empty input in user2Task")
		}

		token, err := Do(c, authTask, None{})
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

	profileTask := NewTask(func(c *Context, input string) (map[string]string, error) {
		recordExecution("profile-start")
		atomic.AddInt32(&profileCounter, 1)

		var userData, user2Data string
		err := Go(c,
			Bind(userTask, input).AssignTo(&userData),
			Bind(user2Task, input+"_alt").AssignTo(&user2Data),
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
	ctx1 := NewContext(context.Background())
	profileResult1, profileErr1 := Do(ctx1, profileTask, testInput1)

	// Execution for second context
	ctx2 := NewContext(context.Background())
	profileResult2, profileErr2 := Do(ctx2, profileTask, testInput2)

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
