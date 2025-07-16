package email

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

var ErrInvalidEmail = errors.New("invalid email format")

type TrimmedOnly string
type Normalized string

type Email struct {
	TrimmedOnly
	Normalized
}

func Normalize(input string) (*Email, error) {
	trimmedOnly := strings.TrimSpace(input)
	if trimmedOnly == "" {
		return nil, ErrInvalidEmail
	}
	if len(trimmedOnly) > 320 {
		return nil, ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(trimmedOnly)
	if err != nil {
		return nil, ErrInvalidEmail
	}
	atIdx := strings.LastIndex(addr.Address, "@")
	if atIdx < 1 || atIdx == len(addr.Address)-1 {
		return nil, ErrInvalidEmail
	}
	local := strings.ToLower(addr.Address[:atIdx])
	localCharsetOK := validateLocalCharset(local)
	if !localCharsetOK {
		return nil, ErrInvalidEmail
	}
	if hasLeadingOrTrailingDotOrDoubleDot(local) {
		return nil, ErrInvalidEmail
	}
	domain, err := idna.Lookup.ToASCII(addr.Address[atIdx+1:])
	if err != nil {
		return nil, ErrInvalidEmail
	}
	if !strings.Contains(domain, ".") || hasLeadingOrTrailingDotOrDoubleDot(domain) {
		return nil, ErrInvalidEmail
	}
	if domain == "gmail.com" || domain == "googlemail.com" {
		local = strings.ReplaceAll(local, ".", "")
	}
	if idx := strings.Index(local, "+"); idx != -1 {
		local = local[:idx]
	}
	if local == "" {
		return nil, ErrInvalidEmail
	}
	return &Email{
		TrimmedOnly: TrimmedOnly(trimmedOnly),
		Normalized:  Normalized(local + "@" + domain),
	}, nil
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
