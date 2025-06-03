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
	p, err := NewProtector(cfg)
	if err != nil {
		t.Fatalf("Failed to create protector: %v", err)
	}
	return p
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
		name    string
		cfg     ProtectorConfig
		wantErr bool
		check   func(*testing.T, *Protector)
	}{
		{
			name: "valid config with defaults",
			cfg: ProtectorConfig{
				GetKeyset: func() *keyset.Keyset { return testKeyset },
			},
			wantErr: false,
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
			wantErr: false,
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
		{
			name: "nil keyset",
			cfg: ProtectorConfig{
				GetKeyset: func() *keyset.Keyset { return nil },
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewProtector(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewProtector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.check != nil {
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
	p, _ := NewProtector(ProtectorConfig{
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
