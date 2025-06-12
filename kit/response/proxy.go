package response

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/river-now/river/kit/htmlutil"
)

// For usage in JSON API handlers that may run in parallel or
// do not have direct access to the http.ResponseWriter.
// Proxy instances are not meant to be shared. Rather, they
// should exist inside a single function/handler scope, and
// afterwards should be used by a parent scope to actually
// de-duplicate, determine priority, and write to the real
// http.ResponseWriter.
type Proxy struct {
	_status      int
	_status_text string
	_headers     map[string][]string
	_cookies     []*http.Cookie
	_head_els    []*htmlutil.Element
	_location    string
}

func NewProxy() *Proxy {
	return &Proxy{_headers: make(map[string][]string)}
}

/////// STATUS (use directly for both success and error responses)

func (p *Proxy) SetStatus(status int, errorStatusText ...string) {
	p._status = status
	if len(errorStatusText) != 0 {
		p._status_text = errorStatusText[0]
	}
}

func (p *Proxy) GetStatus() (int, string) {
	return p._status, p._status_text
}

/////// HEADERS

func (p *Proxy) SetHeader(key, value string) {
	p._headers[key] = []string{value}
}

func (p *Proxy) AddHeader(key, value string) {
	p._headers[key] = append(p._headers[key], value)
}

func (p *Proxy) GetHeader(key string) string {
	if len(p._headers[key]) == 0 {
		return ""
	}
	return p._headers[key][0]
}

func (p *Proxy) GetHeaders(key string) []string {
	return p._headers[key]
}

/////// COOKIES

func (p *Proxy) SetCookie(cookie *http.Cookie) {
	p._cookies = append(p._cookies, cookie)
}

func (p *Proxy) GetCookies() []*http.Cookie {
	return p._cookies
}

/////// HEAD ELEMENTS

func (p *Proxy) AddHeadElement(el *htmlutil.Element) {
	p._head_els = append(p._head_els, el)
}

func (p *Proxy) AddHeadElements(els ...*htmlutil.Element) {
	p._head_els = append(p._head_els, els...)
}

func (p *Proxy) GetHeadElements() []*htmlutil.Element {
	return p._head_els
}

/////// REDIRECTS

// bool return value indicates whether the redirect upgraded to a client redirect
func (p *Proxy) Redirect(r *http.Request, url string, code ...int) bool {
	if doesAcceptClientRedirect(r) {
		p.clientRedirect(url)
		return true
	}
	p.serverRedirect(url, resolveSpreadCode(code))
	return false
}

func (p *Proxy) serverRedirect(url string, code ...int) {
	p._status = resolveSpreadCode(code)
	p._location = url
}

// Use this when you have a client that is initiating requests
// using window.fetch or similar and you want to redirect to
// an external URL. This is a workaround to inherent browser
// limitations with cross-origin redirects in response to a
// an ajax request. For this to actually do anything, you
// need to have a cooperative client that looks for
// X-Client-Redirect headers and manually handles redirects
// using `window.location.href = headerValue` or similar.
// Sets status to 200 if not already set.
func (p *Proxy) clientRedirect(url string) error {
	if ok := validateURL(url); !ok {
		return fmt.Errorf("invalid URL: %s", url)
	}
	currentStatus := p._status
	if currentStatus == 0 {
		p.SetStatus(http.StatusOK)
	}
	p.AddHeader(ClientRedirectHeader, url)
	return nil
}

func (p *Proxy) GetLocation() string {
	return p._location
}

/////// HELPERS

func isError(status int) bool {
	return status >= 400
}

func isServerRedirect(status int) bool {
	return status >= 300 && status < 400
}

func isSuccess(status int) bool {
	return status >= 200 && status < 300
}

func (p *Proxy) IsError() bool {
	return isError(p._status)
}

func (p *Proxy) IsRedirect() bool {
	return p.isServerRedirect() || p.isClientRedirect()
}

func (p *Proxy) isServerRedirect() bool {
	return isServerRedirect(p._status) && p._location != ""
}

func (p *Proxy) isClientRedirect() bool {
	return p.GetHeader(ClientRedirectHeader) != ""
}

func (p *Proxy) IsSuccess() bool {
	return isSuccess(p._status)
}

func (p *Proxy) ApplyToResponseWriter(w http.ResponseWriter, r *http.Request) {
	// Headers
	for k, vs := range p._headers {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}

	// Cookies
	for _, c := range p._cookies {
		http.SetCookie(w, c)
	}

	// Status
	if p._status != 0 {
		if isError(p._status) {
			if p._status_text != "" {
				http.Error(w, p._status_text, p._status)
			} else {
				http.Error(w, http.StatusText(p._status), p._status)
			}
		} else {
			w.WriteHeader(p._status)
		}
	}

	// Redirect
	if p.isServerRedirect() {
		http.Redirect(w, r, p._location, p._status)
	}
}

type cookieWithIdx struct {
	idx    int
	cookie *http.Cookie
}

// Consumers should deduplicate head els after calling MergeProxyResponses
// by using headels.ToHeadEls(proxy.GetHeadElements())
func MergeProxyResponses(proxies ...*Proxy) *Proxy {
	merged := NewProxy()

	// Head Elements -- MERGED IN ORDER
	merged._head_els = make([]*htmlutil.Element, 0)
	for _, p := range proxies {
		merged._head_els = append(merged._head_els, p._head_els...)
	}

	// Headers -- MERGED IN ORDER
	merged._headers = make(map[string][]string)
	for _, p := range proxies {
		for k, vs := range p._headers {
			merged._headers[k] = append(merged._headers[k], vs...)
		}
	}

	// Cookies -- MERGED IN ORDER (later cookies overwrite earlier ones with same name)
	_unique_cookies_map := make(map[string]*cookieWithIdx)
	for i, p := range proxies {
		for _, c := range p._cookies {
			_unique_cookies_map[c.Name] = &cookieWithIdx{i, c}
		}
	}

	deduped := make([]*cookieWithIdx, 0, len(_unique_cookies_map))
	for _, c := range _unique_cookies_map {
		deduped = append(deduped, c)
	}
	slices.SortStableFunc(deduped, func(i, j *cookieWithIdx) int {
		return i.idx - j.idx
	})

	merged._cookies = make([]*http.Cookie, 0, len(deduped))
	for _, c := range deduped {
		merged._cookies = append(merged._cookies, c.cookie)
	}

	// Status
	// Either FIRST ERROR or LAST SUCCESS will win
	for _, p := range proxies {
		if p._status >= 400 { // Error status codes
			merged._status = p._status
			merged._status_text = p._status_text
			break // Take the first error we find
		} else if merged._status < 300 { // Only overwrite success codes
			merged._status = p._status
			merged._status_text = p._status_text
		}
	}

	// Redirect -- Assuming no error, FIRST REDIRECT WINS
	if !isError(merged._status) {
		for _, p := range proxies {
			if p.IsRedirect() {
				merged._status = p._status
				merged._location = p._location
				break
			}
		}
	}

	return merged
}
