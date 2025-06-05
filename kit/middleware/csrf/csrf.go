// Package csrf provides a robust, stateless, and layered CSRF protection middleware for Go.
// It implements the Double Submit Cookie pattern using AEAD-encrypted, HostOnly tokens,
// enhanced with defense-in-depth measures including Origin/Referer validation and session
// binding. Unlike some CSRF prevention patterns, this middleware works regardless of whether
// any user session exists, meaning it also protects pre-authentication POST-ish endpoints
// such as login and registration endpoints.
package csrf

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/river-now/river/kit/bytesutil"
	"github.com/river-now/river/kit/keyset"
	"github.com/river-now/river/kit/netutil"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/securestring"
)

const nonceSize = 24 // Size, in bytes, of the random nonce used in the CSRF token payload.

type payload struct {
	Nonce     []byte    `json:"n"`
	ExpiresAt time.Time `json:"e"`
	SessionID string    `json:"s,omitempty"`
}

func (p payload) isValid() bool {
	return len(p.Nonce) > 0 && !p.ExpiresAt.IsZero() && time.Now().Before(p.ExpiresAt)
}

type ProtectorConfig struct {
	GetKeyset      func() *keyset.Keyset
	AllowedOrigins []string
	TokenTTL       time.Duration
	CookieSuffix   string // Final cookie name will be "__Host-{CookieSuffix}". Defaults to "csrf_token".
	HeaderName     string // Defaults to "X-CSRF-Token"
	// Optional. If non-nil and returns true, the "__Host-" prefix will be omitted and the cookie will
	// not be set as "Secure" or "Partitioned". This is useful for non-HTTPS development environments.
	GetIsDev func() bool
}

type Protector struct {
	cfg                   ProtectorConfig
	isDev                 bool
	cookieName            string
	allowedOrigins        map[string]bool
	hasOriginRestrictions bool
}

func NewProtector(cfg ProtectorConfig) *Protector {
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 4 * time.Hour
	}
	if cfg.CookieSuffix == "" {
		cfg.CookieSuffix = "csrf_token"
	}
	if cfg.HeaderName == "" {
		cfg.HeaderName = "X-CSRF-Token"
	}
	var isDev bool
	if cfg.GetIsDev != nil {
		isDev = cfg.GetIsDev()
	}
	cookieNamePrefix := "__Host-"
	if isDev {
		cookieNamePrefix = "__Dev-"
	}
	cookieName := cookieNamePrefix + cfg.CookieSuffix

	normalized := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil {
			panic(fmt.Sprintf("csrf: invalid origin %q: %v", origin, err))
		}
		if u.Scheme == "" || u.Host == "" {
			panic(fmt.Sprintf("csrf: origin must have scheme and host: %q", origin))
		}
		normalizedOrigin := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
		normalized[normalizedOrigin] = true
	}

	return &Protector{
		cfg:                   cfg,
		isDev:                 isDev,
		cookieName:            cookieName,
		allowedOrigins:        normalized,
		hasOriginRestrictions: len(normalized) > 0,
	}
}

func (p *Protector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p.isDev && !netutil.IsLocalhost(r.Host) {
			panic(fmt.Sprintf(
				"DANGER: CSRF middleware is configured for development mode but the request host is not localhost: %s",
				r.Host,
			))
		}
		if p.isGETLike(r.Method) {
			rp := response.NewProxy()
			if err := p.issueCSRFTokenIfNeeded(rp, r); err != nil {
				log.Printf("csrf.Protector.Middleware: issueCSRFTokenIfNeeded failed: %v\n", err)
			}
			rp.ApplyToResponseWriter(w, r)
			next.ServeHTTP(w, r)
			return
		}
		if err := p.applyCSRFProtection(r); err != nil {
			http.Error(w, "Forbidden: CSRF validation failed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CycleToken generates a new CSRF token and sets it as a cookie.
// Must be called on login (with sessionID) and logout (with empty sessionID).
func (p *Protector) CycleToken(rp *response.Proxy, sessionID string) error {
	token, err := p.newEncryptedPayload(sessionID)
	if err != nil {
		return fmt.Errorf("csrf: failed to generate token: %w", err)
	}
	rp.SetCookie(&http.Cookie{
		Name:        p.cookieName,
		Value:       string(token),
		Secure:      !p.isDev, // Secure if not in dev mode
		SameSite:    http.SameSiteLaxMode,
		Path:        "/",
		MaxAge:      int(p.cfg.TokenTTL.Seconds()),
		Expires:     time.Now().Add(p.cfg.TokenTTL),
		Partitioned: !p.isDev, // Partitioned if not in dev mode
		HttpOnly:    false,    // Must be readable by JavaScript
		Domain:      "",       // Intentionally empty so we can use the __Host- prefix
	})
	return nil
}

// ValidateTokenForSession checks if the CSRF token is valid for the given session.
// Must be called by authentication middleware or handlers for session-bound validation.
func (p *Protector) ValidateTokenForSession(r *http.Request, sessionID string) bool {
	cookie, err := r.Cookie(p.cookieName)
	if err != nil {
		return false
	}
	payload, err := p.decodeEncryptedValue(cookie.Value)
	if err != nil {
		return false
	}
	if !payload.isValid() {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(payload.SessionID), []byte(sessionID)) == 1
}

func (p *Protector) issueCSRFTokenIfNeeded(rp *response.Proxy, r *http.Request) error {
	cookie, err := r.Cookie(p.cookieName)
	if err == nil {
		payload, err := p.decodeEncryptedValue(cookie.Value)
		if err == nil && payload.isValid() {
			return nil
		}
	}
	return p.CycleToken(rp, "")
}

func (p *Protector) applyCSRFProtection(r *http.Request) error {
	if err := p.validateOrigin(r); err != nil {
		return fmt.Errorf("origin validation failed: %w", err)
	}
	cookie, err := r.Cookie(p.cookieName)
	if err != nil {
		return errors.New("csrf token cookie missing")
	}
	payload, err := p.decodeEncryptedValue(cookie.Value)
	if err != nil {
		return fmt.Errorf("invalid csrf token: %w", err)
	}
	if !payload.isValid() {
		return errors.New("csrf token invalid or expired")
	}
	submittedValue := r.Header.Get(p.cfg.HeaderName)
	if submittedValue == "" {
		return errors.New("csrf token missing from request")
	}
	if subtle.ConstantTimeCompare([]byte(submittedValue), []byte(cookie.Value)) != 1 {
		return errors.New("csrf token mismatch")
	}
	return nil
}

func (p *Protector) validateOrigin(r *http.Request) error {
	if !p.hasOriginRestrictions {
		return nil
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		return p.validateOriginHeader(origin, "Origin")
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return p.validateOriginHeader(referer, "Referer")
	}
	return nil
}

func (p *Protector) validateOriginHeader(hdr, label string) error {
	u, err := url.Parse(hdr)
	if err != nil {
		return fmt.Errorf("malformed %s header: %w", label, err)
	}
	origin := strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
	if p.allowedOrigins[origin] {
		return nil
	}
	return fmt.Errorf("%s not allowed: %s", label, origin)
}

func (p *Protector) newEncryptedPayload(sessionID string) (securestring.SecureString, error) {
	nonce, err := bytesutil.Random(nonceSize)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	payload := payload{
		Nonce:     nonce,
		ExpiresAt: time.Now().Add(p.cfg.TokenTTL),
		SessionID: sessionID,
	}
	return securestring.Serialize(p.cfg.GetKeyset(), payload)
}

func (p *Protector) isGETLike(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func (p *Protector) decodeEncryptedValue(v string) (payload, error) {
	return securestring.Deserialize[payload](p.cfg.GetKeyset(), securestring.SecureString(v))
}
