package mux

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// --- TestTaskMiddleware_Interactions ---
func TestTaskMiddleware_Interactions(t *testing.T) {
	t.Run("ErrorFromTaskMiddlewareSetsProxyAndHalts", func(t *testing.T) {
		r := NewRouter(nil)
		var taskMwRan bool
		var mainHandlerRan bool

		taskMw := TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			taskMwRan = true
			rd.ResponseProxy().SetStatus(http.StatusForbidden, "Forbidden by Task MW")
			return None{}, errors.New("task middleware intentional error") // Task also returns an error
		})
		SetGlobalTaskMiddleware(r, taskMw)

		RegisterHandlerFunc(r, http.MethodGet, "/test", func(w http.ResponseWriter, r *http.Request) {
			mainHandlerRan = true
			t.Error("Main handler should not be called if task middleware errors and sets proxy")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if !taskMwRan {
			t.Error("Task middleware should have run")
		}
		if mainHandlerRan {
			t.Error("Main handler ran but should have been short-circuited by task middleware error")
		}
		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
		}
		if !strings.Contains(w.Body.String(), "Forbidden by Task MW") {
			t.Errorf("Expected body to contain 'Forbidden by Task MW', got %q", w.Body.String())
		}
	})

	t.Run("TaskMiddlewareSetsClientErrorAndHalts", func(t *testing.T) {
		r := NewRouter(nil)
		taskMw := TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			rd.ResponseProxy().SetStatus(http.StatusTeapot) // 418 is an error (>=400)
			rd.ResponseProxy().SetHeader("X-Tea-Type", "Earl Grey")
			return None{}, nil // Task itself doesn't return an error, but proxy is set to an error status
		})
		SetGlobalTaskMiddleware(r, taskMw)

		mainHandlerRan := false
		RegisterHandlerFunc(r, http.MethodGet, "/tea", func(w http.ResponseWriter, r *http.Request) {
			mainHandlerRan = true
		})

		req := httptest.NewRequest(http.MethodGet, "/tea", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// CORRECTED ASSERTION: Main handler SHOULD NOT run if task middleware sets a 4xx status
		if mainHandlerRan {
			t.Error("Main handler ran but should have been short-circuited by task middleware 418 error status")
		}
		if w.Code != http.StatusTeapot {
			t.Errorf("Expected status %d, got %d", http.StatusTeapot, w.Code)
		}
		if w.Header().Get("X-Tea-Type") != "Earl Grey" {
			t.Errorf("Expected header 'X-Tea-Type: Earl Grey', got %q", w.Header().Get("X-Tea-Type"))
		}
	})

	t.Run("MultipleTaskMiddlewaresMergeProxiesAndCanHalt", func(t *testing.T) {
		r := NewRouter(nil)
		var mw1Ran, mw2Ran, mw3Ran bool
		var mainHandlerRan bool
		var wg sync.WaitGroup
		wg.Add(3) // For the three task middlewares

		tmw1 := TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			defer wg.Done()
			mw1Ran = true
			rd.ResponseProxy().AddHeader("X-Multi-Trace", "MW1")
			rd.ResponseProxy().SetStatus(http.StatusAccepted)
			return None{}, nil
		})
		// This middleware will cause the halt
		tmw2 := TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			defer wg.Done()
			mw2Ran = true
			rd.ResponseProxy().AddHeader("X-Multi-Trace", "MW2")
			rd.ResponseProxy().SetStatus(http.StatusConflict) // 409 is an error (>=400)
			return None{}, nil
		})
		tmw3 := TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			defer wg.Done()
			mw3Ran = true
			rd.ResponseProxy().AddHeader("X-Multi-Trace", "MW3")
			return None{}, nil
		})

		SetGlobalTaskMiddleware(r, tmw1)
		SetMethodLevelTaskMiddleware(r, http.MethodGet, tmw2)
		route := RegisterHandlerFunc(r, http.MethodGet, "/multi", func(w http.ResponseWriter, r *http.Request) {
			mainHandlerRan = true
		})
		SetPatternLevelTaskMiddleware(route, tmw3)

		req := httptest.NewRequest(http.MethodGet, "/multi", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		wg.Wait()

		if !(mw1Ran && mw2Ran && mw3Ran) {
			t.Errorf("Expected all task middlewares to run, got: mw1=%v, mw2=%v, mw3=%v", mw1Ran, mw2Ran, mw3Ran)
		}
		if mainHandlerRan {
			t.Error("Main handler ran but should have been short-circuited by task middleware 409 error status")
		}
		if w.Code != http.StatusConflict {
			t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
		}
		traces := w.Header().Values("X-Multi-Trace")
		// Order of AddHeader can be non-deterministic for parallel tasks if underlying map write is not guarded
		// For testing, we just check presence and count. response.MergeProxyResponses handles combining them.
		if len(traces) != 3 {
			t.Errorf("Expected 3 X-Multi-Trace headers, got %d: %v", len(traces), traces)
		}
	})
}

// --- TestComplexMiddlewareScenarios ---
// This test should still pass as is, because no task middleware sets an error/redirect status.
func TestComplexMiddlewareScenarios(t *testing.T) {
	t.Run("MixedStackOrderAndExecution", func(t *testing.T) {
		r := NewRouter(nil)
		var executionOrder []string
		var mu sync.Mutex

		appendOrder := func(id string) {
			mu.Lock()
			defer mu.Unlock()
			executionOrder = append(executionOrder, id)
		}

		SetGlobalHTTPMiddleware(r, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				appendOrder("GlobalHTTP-Pre")
				next.ServeHTTP(w, r)
				appendOrder("GlobalHTTP-Post")
			})
		})
		SetMethodLevelHTTPMiddleware(r, http.MethodGet, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				appendOrder("MethodHTTP-Pre")
				next.ServeHTTP(w, r)
				appendOrder("MethodHTTP-Post")
			})
		})

		var globalTaskDone, methodTaskDone, patternTaskDone bool
		var wg sync.WaitGroup
		wg.Add(3)

		SetGlobalTaskMiddleware(r, TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			defer wg.Done()
			appendOrder("GlobalTask")
			rd.ResponseProxy().SetHeader("X-Global-Task", "Done")
			globalTaskDone = true
			return None{}, nil
		}))
		SetMethodLevelTaskMiddleware(r, http.MethodGet, TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			defer wg.Done()
			appendOrder("MethodTask")
			rd.ResponseProxy().SetHeader("X-Method-Task", "Done")
			methodTaskDone = true
			return None{}, nil
		}))

		route := RegisterTaskHandler(r, http.MethodGet, "/complex", TaskHandlerFromFunc(func(rd *ReqData[None]) (string, error) {
			appendOrder("TaskHandler")
			rd.ResponseProxy().SetHeader("X-Handler-Task", "Done")
			return "handler_done", nil
		}))
		SetPatternLevelHTTPMiddleware(route, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				appendOrder("PatternHTTP-Pre")
				next.ServeHTTP(w, r)
				appendOrder("PatternHTTP-Post")
			})
		})
		SetPatternLevelTaskMiddleware(route, TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) {
			defer wg.Done()
			appendOrder("PatternTask")
			rd.ResponseProxy().SetHeader("X-Pattern-Task", "Done")
			patternTaskDone = true
			return None{}, nil
		}))

		req := httptest.NewRequest(http.MethodGet, "/complex", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		wg.Wait()

		expectedHTTPOrder := []string{
			"GlobalHTTP-Pre", "MethodHTTP-Pre", "PatternHTTP-Pre",
			"TaskHandler",
			"PatternHTTP-Post", "MethodHTTP-Post", "GlobalHTTP-Post",
		}
		httpOrder := []string{}
		mu.Lock()
		for _, entry := range executionOrder {
			if strings.Contains(entry, "HTTP") || strings.Contains(entry, "Handler") {
				httpOrder = append(httpOrder, entry)
			}
		}
		mu.Unlock()

		if !sliceEqual(httpOrder, expectedHTTPOrder) {
			t.Errorf("HTTP execution order incorrect.\nExpected: %v\nGot:      %v", expectedHTTPOrder, httpOrder)
		}

		if !(globalTaskDone && methodTaskDone && patternTaskDone) {
			t.Errorf("Not all task middlewares ran: G=%v, M=%v, P=%v", globalTaskDone, methodTaskDone, patternTaskDone)
		}

		if w.Header().Get("X-Global-Task") != "Done" {
			t.Error("Missing X-Global-Task header")
		}
		if w.Header().Get("X-Method-Task") != "Done" {
			t.Error("Missing X-Method-Task header")
		}
		if w.Header().Get("X-Pattern-Task") != "Done" {
			t.Error("Missing X-Pattern-Task header")
		}
		if w.Header().Get("X-Handler-Task") != "Done" {
			t.Error("Missing X-Handler-Task header")
		}
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}
	})
}

// --- TestReqDataAccess ---
// These tests should remain valid as they test fundamental data access.
func TestReqDataAccess(t *testing.T) {
	t.Run("InTaskHandler", func(t *testing.T) {
		r := NewRouter(nil)
		var (
			paramsChecked, splatChecked, tasksCtxChecked, requestChecked, proxyChecked bool
		)
		RegisterTaskHandler(r, http.MethodGet, "/task/:id/path/*", TaskHandlerFromFunc(func(rd *ReqData[None]) (string, error) {
			if len(rd.Params()) > 0 && rd.Params()["id"] == "123" {
				paramsChecked = true
			}
			if len(rd.SplatValues()) > 0 && rd.SplatValues()[0] == "foo" {
				splatChecked = true
			}
			if rd.TasksCtx() != nil {
				tasksCtxChecked = true
			}
			if rd.Request() != nil {
				requestChecked = true
			}
			if rd.ResponseProxy() != nil {
				proxyChecked = true
				rd.ResponseProxy().SetHeader("X-From-Task", "OK")
			}
			return "ok", nil
		}))

		req := httptest.NewRequest(http.MethodGet, "/task/123/path/foo/bar", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if !paramsChecked {
			t.Error("Params not checked or incorrect")
		}
		if !splatChecked {
			t.Error("SplatValues not checked or incorrect")
		}
		if !tasksCtxChecked {
			t.Error("TasksCtx was nil")
		}
		if !requestChecked {
			t.Error("Request was nil")
		}
		if !proxyChecked {
			t.Error("ResponseProxy was nil")
		}
		if w.Header().Get("X-From-Task") != "OK" {
			t.Error("Header from task via proxy not set")
		}
		if w.Code != http.StatusOK { // Added check for OK status
			t.Errorf("Expected status OK for InTaskHandler, got %d", w.Code)
		}
	})

	t.Run("InHTTPHandler_StandardPath", func(t *testing.T) {
		r := NewRouter(nil)
		var paramsChecked, splatChecked bool
		SetGlobalTaskMiddleware(r, TaskMiddlewareFromFunc(func(rd *ReqData[None]) (None, error) { return None{}, nil }))

		RegisterHandlerFunc(r, http.MethodGet, "/http/:id/path/*", func(w http.ResponseWriter, req *http.Request) {
			params := GetParams(req)
			splats := GetSplatValues(req)
			if len(params) > 0 && params["id"] == "456" {
				paramsChecked = true
			}
			if len(splats) > 0 && splats[0] == "baz" {
				splatChecked = true
			}
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/http/456/path/baz/qux", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if !paramsChecked {
			t.Error("Params not checked or incorrect in HTTP handler (standard path)")
		}
		if !splatChecked {
			t.Error("SplatValues not checked or incorrect in HTTP handler (standard path)")
		}
	})
}

// --- TestRoutingEdgeCases ---
// These tests should remain valid.
func TestRoutingEdgeCases(t *testing.T) {
	t.Run("StaticVsParam", func(t *testing.T) {
		r := NewRouter(nil)
		var staticCalled, paramCalled bool
		RegisterHandlerFunc(r, http.MethodGet, "/users/new", func(w http.ResponseWriter, r *http.Request) { staticCalled = true })
		RegisterHandlerFunc(r, http.MethodGet, "/users/:id", func(w http.ResponseWriter, r *http.Request) { paramCalled = true })

		req := httptest.NewRequest(http.MethodGet, "/users/new", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		if !staticCalled || paramCalled {
			t.Errorf("Expected static route /users/new to be called, staticCalled=%v, paramCalled=%v", staticCalled, paramCalled)
		}

		staticCalled, paramCalled = false, false
		req = httptest.NewRequest(http.MethodGet, "/users/123", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		if staticCalled || !paramCalled {
			t.Errorf("Expected param route /users/:id to be called, staticCalled=%v, paramCalled=%v", staticCalled, paramCalled)
		}
	})

	t.Run("SplatURLDecoding", func(t *testing.T) {
		r := NewRouter(nil)
		var capturedSplatValues []string

		// Use a splat pattern
		RegisterHandlerFunc(r, http.MethodGet, "/test/*", func(w http.ResponseWriter, req *http.Request) {
			capturedSplatValues = GetSplatValues(req)
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test/foo%20bar%2Fbaz", nil)
		// r.URL.Path becomes "/test/foo bar/baz"
		// ParseSegments of "test/foo bar/baz" yields ["test", "foo bar", "baz"]
		// The splat for "/test/*" should capture ["foo bar", "baz"]

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Expected status OK, got %d", rec.Code)
		}

		expectedSplat := []string{"foo bar", "baz"}
		if !sliceEqual(capturedSplatValues, expectedSplat) {
			t.Errorf("Expected splat values %v, got %v", expectedSplat, capturedSplatValues)
		}

		// To get the string "foo bar/baz", the handler/test would join:
		reconstructedValue := strings.Join(capturedSplatValues, "/")
		if reconstructedValue != "foo bar/baz" {
			t.Errorf("Expected reconstructed value to be 'foo bar/baz', got %q", reconstructedValue)
		}
	})

	t.Run("EmptySplat", func(t *testing.T) {
		r := NewRouter(nil)
		var splatValues []string
		RegisterHandlerFunc(r, http.MethodGet, "/files/*", func(w http.ResponseWriter, req *http.Request) {
			splatValues = GetSplatValues(req)
		})
		req := httptest.NewRequest(http.MethodGet, "/files/", nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
		if !(len(splatValues) == 1 && splatValues[0] == "") {
			t.Errorf("Expected splat for /files/ to be [\"\"], got %v", splatValues)
		}
	})
}

// --- TestServeHTTP_ErrorHandling ---
func TestServeHTTP_ErrorHandling(t *testing.T) {
	t.Run("PanicRecoveryMiddleware", func(t *testing.T) {
		r := NewRouter(nil)
		SetGlobalHTTPMiddleware(r, func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if err := recover(); err != nil {
						fmt.Println("Recovered from panic:", err)
						http.Error(w, "Recovered Internal Server Error", http.StatusInternalServerError)
					}
				}()
				next.ServeHTTP(w, r)
			})
		})
		RegisterHandlerFunc(r, http.MethodGet, "/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("intentional panic in handler")
		})
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 after panic, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Recovered Internal Server Error") {
			t.Errorf("Expected recovery message in body, got %q", w.Body.String())
		}
	})

	t.Run("NilTaskHandlerLeadsToError", func(t *testing.T) {
		r := NewRouter(nil)
		var nilTask *TaskHandler[None, None]
		_ = RegisterTaskHandler(r, http.MethodGet, "/nil-task", nilTask)
		req := httptest.NewRequest(http.MethodGet, "/nil-task", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError { // Expecting 500 due to robust check in createTaskFinalHandler
			t.Errorf("Expected non-OK status (specifically 500) for nil task handler, got %d", w.Code)
		}
	})
}

// --- TestMarshalInputEdgeCases ---
// These tests should remain valid.
func TestMarshalInputEdgeCases(t *testing.T) {
	type MyInput struct {
		Field string `json:"field"`
	}
	type MyOutput struct {
		OutputField string `json:"outputField"`
	}

	t.Run("NilMarshalInputWithTaskExpectingInput", func(t *testing.T) {
		r := NewRouter(&Options{MarshalInput: nil})
		var receivedInput MyInput
		RegisterTaskHandler(r, http.MethodPost, "/test", TaskHandlerFromFunc(func(rd *ReqData[MyInput]) (MyOutput, error) {
			receivedInput = rd.Input()
			return MyOutput{OutputField: "got: " + rd.Input().Field}, nil
		}))
		body := strings.NewReader(`{"field":"hello"}`)
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}
		if receivedInput.Field != "" {
			t.Errorf("Expected zero value for input field, got %q", receivedInput.Field)
		}
	})

	t.Run("MarshalInputMutatesInputPtr", func(t *testing.T) {
		r := NewRouter(&Options{
			MarshalInput: func(req *http.Request, inputPtr any) error {
				if err := json.NewDecoder(req.Body).Decode(inputPtr); err != nil {
					return err
				}
				if mi, ok := inputPtr.(*MyInput); ok {
					mi.Field += "_mutated"
				}
				return nil
			},
		})
		var receivedInput MyInput
		RegisterTaskHandler(r, http.MethodPost, "/test", TaskHandlerFromFunc(func(rd *ReqData[MyInput]) (MyOutput, error) {
			receivedInput = rd.Input()
			return MyOutput{OutputField: "final: " + rd.Input().Field}, nil
		}))
		body := strings.NewReader(`{"field":"original"}`)
		req := httptest.NewRequest(http.MethodPost, "/test", body)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}
		if receivedInput.Field != "original_mutated" {
			t.Errorf("Expected mutated input 'original_mutated', got %q", receivedInput.Field)
		}
	})
}
