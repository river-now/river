package email

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

var ErrInvalidEmail = errors.New("invalid email format")

type Normalized string

func Normalize(input string) (Normalized, error) {
	e := strings.TrimSpace(input)
	if e == "" {
		return "", ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(e)
	if err != nil {
		return "", ErrInvalidEmail
	}
	atIdx := strings.LastIndex(addr.Address, "@")
	if atIdx < 1 || atIdx == len(addr.Address)-1 {
		return "", ErrInvalidEmail
	}
	local := strings.ToLower(addr.Address[:atIdx])
	localCharsetOK := validateLocalCharset(local)
	if !localCharsetOK {
		return "", ErrInvalidEmail
	}
	if hasLeadingOrTrailingDotOrDoubleDot(local) {
		return "", ErrInvalidEmail
	}
	domain, err := idna.Lookup.ToASCII(addr.Address[atIdx+1:])
	if err != nil {
		return "", ErrInvalidEmail
	}
	if !strings.Contains(domain, ".") || hasLeadingOrTrailingDotOrDoubleDot(domain) {
		return "", ErrInvalidEmail
	}
	if domain == "gmail.com" || domain == "googlemail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	if idx := strings.Index(local, "+"); idx != -1 {
		local = local[:idx]
	}
	if local == "" {
		return "", ErrInvalidEmail
	}
	return Normalized(local + "@" + domain), nil
}

func validateLocalCharset(input string) bool {
	for _, r := range input {
		if r > unicode.MaxASCII {
			return false
		}
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.', r == '_', r == '%', r == '+', r == '-':
		default:
			return false
		}
	}
	return true
}

func hasLeadingOrTrailingDotOrDoubleDot(input string) bool {
	if len(input) == 0 {
		return false
	}
	if strings.Contains(input, "..") {
		return true
	}
	return input[0] == '.' || input[len(input)-1] == '.'
}
