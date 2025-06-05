package csrf

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/river-now/river/kit/keyset"
	"github.com/river-now/river/kit/response"
)

// Test helpers
func createTestKeyset(t *testing.T) *keyset.Keyset {
	// Create a valid base64-encoded 32-byte secret
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	b64Secret := base64.StdEncoding.EncodeToString(secret)

	ks, err := keyset.RootSecretsToRootKeyset(keyset.RootSecrets{keyset.RootSecret(b64Secret)})
	if err != nil {
		t.Fatalf("Failed to create test keyset: %v", err)
	}
	return ks
}

func createTestProtector(t *testing.T, origins []string) *Protector {
	testKeyset := createTestKeyset(t)
	cfg := ProtectorConfig{
		GetKeyset:      func() *keyset.Keyset { return testKeyset },
		AllowedOrigins: origins,
		TokenTTL:       1 * time.Hour,
	}
	return NewProtector(cfg)
}

func extractCSRFCookie(rr *httptest.ResponseRecorder, cookieName string) *http.Cookie {
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == cookieName {
			return cookie
		}
	}
	return nil
}

func extractTokenFromCookie(cookie *http.Cookie) string {
	return cookie.Value
}

// Tests
func TestNewProtector(t *testing.T) {
	testKeyset := createTestKeyset(t)

	tests := []struct {
		name  string
		cfg   ProtectorConfig
		check func(*testing.T, *Protector)
	}{
		{
			name: "valid config with defaults",
			cfg: ProtectorConfig{
				GetKeyset: func() *keyset.Keyset { return testKeyset },
			},
			check: func(t *testing.T, p *Protector) {
				if p.cfg.TokenTTL != 4*time.Hour {
					t.Errorf("Expected default TTL of 4h, got %v", p.cfg.TokenTTL)
				}
				if p.cfg.CookieSuffix != "csrf_token" {
					t.Errorf("Expected default cookie suffix 'csrf_token', got %s", p.cfg.CookieSuffix)
				}
				if p.cfg.HeaderName != "X-CSRF-Token" {
					t.Errorf("Expected default header name 'X-CSRF-Token', got %s", p.cfg.HeaderName)
				}
				if p.cookieName != "__Host-csrf_token" {
					t.Errorf("Expected cookie name '__Host-csrf_token', got %s", p.cookieName)
				}
			},
		},
		{
			name: "custom values",
			cfg: ProtectorConfig{
				GetKeyset:      func() *keyset.Keyset { return testKeyset },
				AllowedOrigins: []string{"https://example.com", "HTTPS://EXAMPLE.ORG"},
				TokenTTL:       2 * time.Hour,
				CookieSuffix:   "custom",
				HeaderName:     "X-Custom-CSRF",
			},
			check: func(t *testing.T, p *Protector) {
				if p.cfg.TokenTTL != 2*time.Hour {
					t.Errorf("Expected TTL of 2h, got %v", p.cfg.TokenTTL)
				}
				if p.cookieName != "__Host-custom" {
					t.Errorf("Expected cookie name '__Host-custom', got %s", p.cookieName)
				}
				if !p.allowedOrigins["https://example.com"] {
					t.Error("Expected normalized origin 'https://example.com' to be allowed")
				}
				if !p.allowedOrigins["https://example.org"] {
					t.Error("Expected normalized origin 'https://example.org' to be allowed")
				}
				if p.hasOriginRestrictions != true {
					t.Error("Expected hasOriginRestrictions to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProtector(tt.cfg)
			if tt.check != nil {
				tt.check(t, p)
			}
		})
	}
}

func TestMiddleware_GETRequest(t *testing.T) {
	p := createTestProtector(t, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name           string
		method         string
		existingCookie *http.Cookie
		wantCookie     bool
		wantStatus     int
	}{
		{
			name:       "GET without existing cookie",
			method:     "GET",
			wantCookie: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "HEAD without existing cookie",
			method:     "HEAD",
			wantCookie: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "OPTIONS without existing cookie",
			method:     "OPTIONS",
			wantCookie: true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "TRACE without existing cookie",
			method:     "TRACE",
			wantCookie: true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.existingCookie != nil {
				req.AddCookie(tt.existingCookie)
			}

			rr := httptest.NewRecorder()
			p.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			cookie := extractCSRFCookie(rr, p.cookieName)
			if tt.wantCookie && cookie == nil {
				t.Error("Expected CSRF cookie to be set")
			}
			if tt.wantCookie && cookie != nil {
				// Verify cookie attributes
				if !cookie.Secure {
					t.Error("Expected Secure flag to be true")
				}
				if cookie.SameSite != http.SameSiteLaxMode {
					t.Errorf("Expected SameSite=Lax, got %v", cookie.SameSite)
				}
				if cookie.HttpOnly {
					t.Error("Expected HttpOnly to be false (must be readable by JS)")
				}
				if cookie.Path != "/" {
					t.Errorf("Expected Path=/, got %s", cookie.Path)
				}
				if cookie.Domain != "" {
					t.Errorf("Expected empty Domain for __Host- prefix, got %s", cookie.Domain)
				}
			}
		})
	}
}

func TestMiddleware_POSTRequest(t *testing.T) {
	p := createTestProtector(t, []string{"https://example.com"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// First, get a valid CSRF token via GET request
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(getRR, getReq)

	cookie := extractCSRFCookie(getRR, p.cookieName)
	if cookie == nil {
		t.Fatal("Failed to get CSRF cookie from GET request")
	}

	// Extract the token to use (which is the cookie value itself)
	token := extractTokenFromCookie(cookie)

	tests := []struct {
		name       string
		method     string
		cookie     *http.Cookie
		token      string
		origin     string
		referer    string
		wantStatus int
	}{
		{
			name:       "valid POST with token and origin",
			method:     "POST",
			cookie:     cookie,
			token:      token,
			origin:     "https://example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid POST with token and referer",
			method:     "POST",
			cookie:     cookie,
			token:      token,
			referer:    "https://example.com/page",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST without cookie",
			method:     "POST",
			token:      token,
			origin:     "https://example.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST without token header",
			method:     "POST",
			cookie:     cookie,
			origin:     "https://example.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST with wrong token",
			method:     "POST",
			cookie:     cookie,
			token:      "wrong-token",
			origin:     "https://example.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST with wrong origin",
			method:     "POST",
			cookie:     cookie,
			token:      token,
			origin:     "https://evil.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "POST with wrong referer",
			method:     "POST",
			cookie:     cookie,
			token:      token,
			referer:    "https://evil.com/page",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "PUT request",
			method:     "PUT",
			cookie:     cookie,
			token:      token,
			origin:     "https://example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE request",
			method:     "DELETE",
			cookie:     cookie,
			token:      token,
			origin:     "https://example.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "PATCH request",
			method:     "PATCH",
			cookie:     cookie,
			token:      token,
			origin:     "https://example.com",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			if tt.token != "" {
				req.Header.Set(p.cfg.HeaderName, tt.token)
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}

			rr := httptest.NewRecorder()
			p.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestMiddleware_NoOriginRestrictions(t *testing.T) {
	p := createTestProtector(t, nil) // No allowed origins

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Get token
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(getRR, getReq)

	cookie := extractCSRFCookie(getRR, p.cookieName)
	token := extractTokenFromCookie(cookie)

	// POST should succeed without origin validation
	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(cookie)
	req.Header.Set(p.cfg.HeaderName, token)
	req.Header.Set("Origin", "https://any-origin.com")

	rr := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestCycleToken(t *testing.T) {
	p := createTestProtector(t, nil)

	tests := []struct {
		name      string
		sessionID string
	}{
		{
			name:      "cycle with session",
			sessionID: "test-session-123",
		},
		{
			name:      "cycle without session",
			sessionID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := response.NewProxy()

			err := p.CycleToken(rp, tt.sessionID)
			if err != nil {
				t.Fatalf("CycleToken failed: %v", err)
			}

			// Apply proxy to response writer to get cookies
			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			rp.ApplyToResponseWriter(rr, req)

			cookie := extractCSRFCookie(rr, p.cookieName)
			if cookie == nil {
				t.Fatal("No cookie set after CycleToken")
			}

			// Decode and verify session ID
			payload, err := p.decodeEncryptedValue(cookie.Value)
			if err != nil {
				t.Fatalf("Failed to decode cycled token: %v", err)
			}

			if payload.SessionID != tt.sessionID {
				t.Errorf("Expected session ID %q, got %q", tt.sessionID, payload.SessionID)
			}
		})
	}
}

func TestValidateTokenForSession(t *testing.T) {
	p := createTestProtector(t, nil)
	sessionID := "test-session-456"

	// Create a token with session
	rp := response.NewProxy()
	err := p.CycleToken(rp, sessionID)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Apply proxy to get cookie
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	rp.ApplyToResponseWriter(rr, req)

	cookie := extractCSRFCookie(rr, p.cookieName)

	tests := []struct {
		name      string
		cookie    *http.Cookie
		sessionID string
		want      bool
	}{
		{
			name:      "valid session",
			cookie:    cookie,
			sessionID: sessionID,
			want:      true,
		},
		{
			name:      "wrong session",
			cookie:    cookie,
			sessionID: "wrong-session",
			want:      false,
		},
		{
			name:      "empty session",
			cookie:    cookie,
			sessionID: "",
			want:      false,
		},
		{
			name:      "no cookie",
			cookie:    nil,
			sessionID: sessionID,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			got := p.ValidateTokenForSession(req, tt.sessionID)
			if got != tt.want {
				t.Errorf("ValidateTokenForSession() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	testKeyset := createTestKeyset(t)
	p := NewProtector(ProtectorConfig{
		GetKeyset: func() *keyset.Keyset { return testKeyset },
		TokenTTL:  100 * time.Millisecond, // Very short TTL for testing
	})

	// Create token
	rp := response.NewProxy()
	err := p.CycleToken(rp, "")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Apply proxy to get cookie
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	rp.ApplyToResponseWriter(rr, req)

	cookie := extractCSRFCookie(rr, p.cookieName)
	token := extractTokenFromCookie(cookie)

	// Immediate validation should succeed
	req = httptest.NewRequest("POST", "/", nil)
	req.AddCookie(cookie)
	req.Header.Set(p.cfg.HeaderName, token)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rr = httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Error("Expected immediate validation to succeed")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Validation should now fail
	rr = httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Error("Expected validation to fail after expiration")
	}
}

func TestGETRequestWithExistingValidToken(t *testing.T) {
	p := createTestProtector(t, nil)

	// First GET to get token
	req1 := httptest.NewRequest("GET", "/", nil)
	rr1 := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	p.Middleware(handler).ServeHTTP(rr1, req1)

	cookie := extractCSRFCookie(rr1, p.cookieName)
	if cookie == nil {
		t.Fatal("No cookie from first GET")
	}

	// Second GET with existing valid cookie
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr2, req2)

	// Should not issue new cookie
	newCookie := extractCSRFCookie(rr2, p.cookieName)
	if newCookie != nil {
		t.Error("Should not issue new cookie when valid one exists")
	}
}

func TestOriginValidationWithMalformedReferer(t *testing.T) {
	p := createTestProtector(t, []string{"https://example.com"})

	// Get token
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	p.Middleware(handler).ServeHTTP(getRR, getReq)

	cookie := extractCSRFCookie(getRR, p.cookieName)
	token := extractTokenFromCookie(cookie)

	// POST with malformed referer
	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(cookie)
	req.Header.Set(p.cfg.HeaderName, token)
	req.Header.Set("Referer", "not-a-valid-url")

	rr := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected %d for malformed referer, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestCookieAttributes(t *testing.T) {
	p := createTestProtector(t, nil)

	rp := response.NewProxy()
	err := p.CycleToken(rp, "")
	if err != nil {
		t.Fatalf("Failed to cycle token: %v", err)
	}

	// Apply proxy to get cookie
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	rp.ApplyToResponseWriter(rr, req)

	cookie := extractCSRFCookie(rr, p.cookieName)
	if cookie == nil {
		t.Fatal("No cookie set")
	}

	// Verify all security-critical attributes
	if !strings.HasPrefix(cookie.Name, "__Host-") {
		t.Errorf("Cookie name must start with __Host-, got %s", cookie.Name)
	}
	if !cookie.Secure {
		t.Error("Cookie must have Secure flag")
	}
	if cookie.Domain != "" {
		t.Error("Cookie must have empty Domain for __Host- prefix")
	}
	if cookie.Path != "/" {
		t.Error("Cookie must have Path=/")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("Cookie must have SameSite=Lax, got %v", cookie.SameSite)
	}
	if cookie.HttpOnly {
		t.Error("Cookie must not be HttpOnly (needs JS access)")
	}
	if !cookie.Partitioned {
		t.Error("Cookie should be Partitioned")
	}
}

// TestDevMode tests the development mode functionality
func TestDevMode(t *testing.T) {
	testKeyset := createTestKeyset(t)

	tests := []struct {
		name        string
		host        string
		shouldPanic bool
	}{
		{
			name:        "localhost allowed",
			host:        "localhost:8080",
			shouldPanic: false,
		},
		{
			name:        "127.0.0.1 allowed",
			host:        "127.0.0.1:3000",
			shouldPanic: false,
		},
		{
			name:        "::1 allowed",
			host:        "[::1]:8080",
			shouldPanic: false,
		},
		{
			name:        "localhost without port allowed",
			host:        "localhost",
			shouldPanic: false,
		},
		{
			name:        "non-localhost should panic",
			host:        "example.com",
			shouldPanic: true,
		},
		{
			name:        "IP address should panic",
			host:        "192.168.1.1:8080",
			shouldPanic: true,
		},
		{
			name:        "subdomain should panic",
			host:        "app.localhost:8080",
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProtector(ProtectorConfig{
				GetKeyset: func() *keyset.Keyset { return testKeyset },
				GetIsDev:  func() bool { return true }, // Enable dev mode
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			rr := httptest.NewRecorder()

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("Expected panic for host %s, but didn't panic", tt.host)
					}
				}()
			}

			p.Middleware(handler).ServeHTTP(rr, req)

			if !tt.shouldPanic {
				// Verify dev mode cookie attributes
				cookie := extractCSRFCookie(rr, p.cookieName)
				if cookie == nil {
					t.Fatal("Expected cookie to be set")
				}

				// In dev mode, should NOT have __Host- prefix
				if cookie.Name != "__Dev-csrf_token" {
					t.Errorf("Expected cookie name 'csrf_token' in dev mode, got %s", cookie.Name)
				}

				// Should NOT be Secure in dev mode
				if cookie.Secure {
					t.Error("Cookie should not be Secure in dev mode")
				}

				// Should NOT be Partitioned in dev mode
				if cookie.Partitioned {
					t.Error("Cookie should not be Partitioned in dev mode")
				}
			}
		})
	}
}

// TestDevModeVsProductionMode compares behavior between modes
func TestDevModeVsProductionMode(t *testing.T) {
	testKeyset := createTestKeyset(t)

	// Test production mode (default)
	prodProtector := NewProtector(ProtectorConfig{
		GetKeyset: func() *keyset.Keyset { return testKeyset },
		// GetIsDev is nil, so production mode
	})

	// Test dev mode
	devProtector := NewProtector(ProtectorConfig{
		GetKeyset: func() *keyset.Keyset { return testKeyset },
		GetIsDev:  func() bool { return true },
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Production mode test
	t.Run("production mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		prodProtector.Middleware(handler).ServeHTTP(rr, req)

		cookie := extractCSRFCookie(rr, prodProtector.cookieName)
		if cookie.Name != "__Host-csrf_token" {
			t.Errorf("Expected __Host- prefix in production, got %s", cookie.Name)
		}
		if !cookie.Secure {
			t.Error("Expected Secure flag in production")
		}
		if !cookie.Partitioned {
			t.Error("Expected Partitioned flag in production")
		}
	})

	// Dev mode test
	t.Run("dev mode", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Host = "localhost:8080"
		rr := httptest.NewRecorder()
		devProtector.Middleware(handler).ServeHTTP(rr, req)

		cookie := extractCSRFCookie(rr, devProtector.cookieName)
		if cookie.Name != "__Dev-csrf_token" {
			t.Errorf("Expected no __Host- prefix in dev mode, got %s", cookie.Name)
		}
		if cookie.Secure {
			t.Error("Expected no Secure flag in dev mode")
		}
		if cookie.Partitioned {
			t.Error("Expected no Partitioned flag in dev mode")
		}
	})
}

// TestOriginValidationEdgeCases tests edge cases in origin validation
func TestOriginValidationEdgeCases(t *testing.T) {
	p := createTestProtector(t, []string{"https://example.com", "http://localhost:3000"})

	// Get a valid token
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	p.Middleware(handler).ServeHTTP(getRR, getReq)

	cookie := extractCSRFCookie(getRR, p.cookieName)
	token := extractTokenFromCookie(cookie)

	tests := []struct {
		name       string
		origin     string
		referer    string
		wantStatus int
	}{
		{
			name:       "no origin or referer with restrictions should pass",
			wantStatus: http.StatusOK,
		},
		{
			name:       "case insensitive origin matching",
			origin:     "HTTPS://EXAMPLE.COM",
			wantStatus: http.StatusOK,
		},
		{
			name:       "case insensitive referer matching",
			referer:    "HTTPS://EXAMPLE.COM/page",
			wantStatus: http.StatusOK,
		},
		{
			name:       "origin takes precedence over referer",
			origin:     "https://example.com",
			referer:    "https://evil.com",
			wantStatus: http.StatusOK,
		},
		{
			name:       "localhost with port allowed",
			origin:     "http://localhost:3000",
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty origin and referer",
			origin:     "",
			referer:    "",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.AddCookie(cookie)
			req.Header.Set(p.cfg.HeaderName, token)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}

			rr := httptest.NewRecorder()
			p.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}

// TestCustomHeaderName tests using a custom header name
func TestCustomHeaderName(t *testing.T) {
	testKeyset := createTestKeyset(t)
	customHeaderName := "X-Custom-CSRF-Token"

	p := NewProtector(ProtectorConfig{
		GetKeyset:  func() *keyset.Keyset { return testKeyset },
		HeaderName: customHeaderName,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Get token
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(getRR, getReq)

	cookie := extractCSRFCookie(getRR, p.cookieName)
	token := extractTokenFromCookie(cookie)

	// POST with custom header
	req := httptest.NewRequest("POST", "/", nil)
	req.AddCookie(cookie)
	req.Header.Set(customHeaderName, token)

	rr := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected %d with custom header, got %d", http.StatusOK, rr.Code)
	}

	// POST with default header name should fail
	req2 := httptest.NewRequest("POST", "/", nil)
	req2.AddCookie(cookie)
	req2.Header.Set("X-CSRF-Token", token) // Using default header name

	rr2 := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusForbidden {
		t.Errorf("Expected %d with wrong header name, got %d", http.StatusForbidden, rr2.Code)
	}
}

// TestInvalidTokenPayload tests handling of corrupted tokens
func TestInvalidTokenPayload(t *testing.T) {
	p := createTestProtector(t, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		tokenValue string
		wantStatus int
	}{
		{
			name:       "empty token",
			tokenValue: "",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid base64",
			tokenValue: "not-valid-base64!@#$",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "truncated token",
			tokenValue: "SGVsbG8=", // Valid base64 but not a valid encrypted payload
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "random data",
			tokenValue: base64.StdEncoding.EncodeToString([]byte("random data that's not encrypted")),
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  p.cookieName,
				Value: tt.tokenValue,
			})
			req.Header.Set(p.cfg.HeaderName, tt.tokenValue)

			rr := httptest.NewRecorder()
			p.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d for %s, got %d", tt.wantStatus, tt.name, rr.Code)
			}
		})
	}
}

// TestConcurrentRequests tests thread safety
func TestConcurrentRequests(t *testing.T) {
	p := createTestProtector(t, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Get a valid token
	getReq := httptest.NewRequest("GET", "/", nil)
	getRR := httptest.NewRecorder()
	p.Middleware(handler).ServeHTTP(getRR, getReq)

	cookie := extractCSRFCookie(getRR, p.cookieName)
	token := extractTokenFromCookie(cookie)

	// Run concurrent POST requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("POST", "/", nil)
			req.AddCookie(cookie)
			req.Header.Set(p.cfg.HeaderName, token)

			rr := httptest.NewRecorder()
			p.Middleware(handler).ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Concurrent request failed with status %d", rr.Code)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestKeysetErrors tests error cases with invalid keysets
func TestKeysetErrors(t *testing.T) {
	// Test with nil keyset
	p1 := NewProtector(ProtectorConfig{
		GetKeyset: func() *keyset.Keyset { return nil },
		TokenTTL:  1 * time.Hour,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// GET request with nil keyset
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	p1.Middleware(handler).ServeHTTP(rr, req)

	// Should handle error and still return 200
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 with nil keyset, got %d", rr.Code)
	}

	// Test with empty keyset
	emptyKeyset := &keyset.Keyset{}
	p2 := NewProtector(ProtectorConfig{
		GetKeyset: func() *keyset.Keyset { return emptyKeyset },
		TokenTTL:  1 * time.Hour,
	})

	req2 := httptest.NewRequest("GET", "/", nil)
	rr2 := httptest.NewRecorder()
	p2.Middleware(handler).ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("Expected status 200 with empty keyset, got %d", rr2.Code)
	}
}

// TestResponseProxyIntegration verifies response proxy usage
func TestResponseProxyIntegration(t *testing.T) {
	p := createTestProtector(t, nil)

	// Custom handler that writes before middleware completes
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write some content
		w.Header().Set("X-Custom-Header", "test")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	p.Middleware(handler).ServeHTTP(rr, req)

	// Verify cookie was set despite handler writing response
	cookie := extractCSRFCookie(rr, p.cookieName)
	if cookie == nil {
		t.Error("Cookie should be set even when handler writes response")
	}

	// Verify custom header is preserved
	if rr.Header().Get("X-Custom-Header") != "test" {
		t.Error("Custom headers should be preserved")
	}

	// Verify body content
	if rr.Body.String() != "test content" {
		t.Error("Body content should be preserved")
	}
}
