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
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/securestring"
)

const NonceSize = 24 // Size, in bytes, of the random nonce used in the CSRF token payload.

type payload struct {
	Nonce     []byte    `json:"n"`
	ExpiresAt time.Time `json:"exp"`
	SessionID string    `json:"sid,omitempty"`
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
}

type Protector struct {
	cfg                   ProtectorConfig
	cookieName            string
	allowedOrigins        map[string]bool
	hasOriginRestrictions bool
}

func NewProtector(cfg ProtectorConfig) (*Protector, error) {
	if err := cfg.GetKeyset().Validate(); err != nil {
		return nil, fmt.Errorf("csrf: invalid keyset: %w", err)
	}
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 4 * time.Hour
	}
	if cfg.CookieSuffix == "" {
		cfg.CookieSuffix = "csrf_token"
	}
	if cfg.HeaderName == "" {
		cfg.HeaderName = "X-CSRF-Token"
	}
	cookieName := "__Host-" + cfg.CookieSuffix
	normalizedOrigins := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		normalizedOrigins[strings.ToLower(origin)] = true
	}
	return &Protector{
		cfg:                   cfg,
		cookieName:            cookieName,
		allowedOrigins:        normalizedOrigins,
		hasOriginRestrictions: len(normalizedOrigins) > 0,
	}, nil
}

func (p *Protector) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// Must be called on login (with sessionID) and logout (with empty sessionID).
func (p *Protector) CycleToken(rp *response.Proxy, sessionID string) error {
	token, err := p.newEncryptedPayload(sessionID)
	if err != nil {
		return fmt.Errorf("csrf: failed to generate token: %w", err)
	}
	rp.SetCookie(&http.Cookie{
		Name:        p.cookieName,
		Value:       string(token),
		Secure:      true,
		SameSite:    http.SameSiteLaxMode,
		Path:        "/",
		MaxAge:      int(p.cfg.TokenTTL.Seconds()),
		Expires:     time.Now().Add(p.cfg.TokenTTL),
		Partitioned: true,
		HttpOnly:    false, // Must be readable by JavaScript
		Domain:      "",    // Intentionally empty so we can use the __Host- prefix
	})
	return nil
}

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
	origin := r.Header.Get("Origin")
	if origin != "" {
		if p.allowedOrigins[strings.ToLower(origin)] {
			return nil
		}
		return errors.New("origin not allowed")
	}
	referer := r.Header.Get("Referer")
	if referer != "" {
		refererURL, err := url.Parse(referer)
		if err != nil {
			return errors.New("malformed referer header")
		}
		refererOrigin := strings.ToLower(fmt.Sprintf("%s://%s", refererURL.Scheme, refererURL.Host))
		if p.allowedOrigins[refererOrigin] {
			return nil
		}
		return errors.New("referer not allowed")
	}
	return nil
}

func (p *Protector) newEncryptedPayload(sessionID string) (securestring.SecureString, error) {
	nonce, err := bytesutil.Random(NonceSize)
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
