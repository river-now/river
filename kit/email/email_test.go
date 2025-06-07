package email

import (
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Normalized
		wantErr bool
	}{
		// Basic valid cases
		{
			name:  "simple valid email",
			input: "user@example.com",
			want:  "user@example.com",
		},
		{
			name:  "email with uppercase letters",
			input: "User@Example.COM",
			want:  "user@example.com",
		},
		{
			name:  "email with whitespace",
			input: "  user@example.com  ",
			want:  "user@example.com",
		},
		{
			name:  "email with display name",
			input: "John Doe <john@example.com>",
			want:  "john@example.com",
		},
		{
			name:  "email with quoted display name",
			input: `"John Doe" <john@example.com>`,
			want:  "john@example.com",
		},

		// Gmail normalization
		{
			name:  "gmail with dots",
			input: "john.doe@gmail.com",
			want:  "johndoe@gmail.com",
		},
		{
			name:  "googlemail with dots",
			input: "john.doe@googlemail.com",
			want:  "johndoe@googlemail.com",
		},
		{
			name:  "gmail with plus and dots",
			input: "john.doe+tag@gmail.com",
			want:  "johndoe@gmail.com",
		},
		{
			name:  "non-gmail with dots preserved",
			input: "john.doe@example.com",
			want:  "john.doe@example.com",
		},

		// Plus addressing
		{
			name:  "email with plus tag",
			input: "user+tag@example.com",
			want:  "user@example.com",
		},
		{
			name:  "email with plus tag and more content",
			input: "user+tag+more@example.com",
			want:  "user@example.com",
		},
		{
			name:  "gmail with plus and dots combined",
			input: "john.doe+newsletter@gmail.com",
			want:  "johndoe@gmail.com",
		},

		// Valid special characters in local part
		{
			name:  "email with underscore",
			input: "user_name@example.com",
			want:  "user_name@example.com",
		},
		{
			name:  "email with hyphen",
			input: "user-name@example.com",
			want:  "user-name@example.com",
		},
		{
			name:  "email with percent",
			input: "user%name@example.com",
			want:  "user%name@example.com",
		},
		{
			name:  "email with multiple valid special chars",
			input: "user.name_test-1%2@example.com",
			want:  "user.name_test-1%2@example.com",
		},

		// Internationalized domain names
		{
			name:  "IDN domain with unicode",
			input: "user@münchen.de",
			want:  "user@xn--mnchen-3ya.de",
		},
		{
			name:  "IDN domain already punycode",
			input: "user@xn--mnchen-3ya.de",
			want:  "user@xn--mnchen-3ya.de",
		},

		// Edge cases for valid emails
		{
			name:  "single char local part",
			input: "a@example.com",
			want:  "a@example.com",
		},
		{
			name:  "numbers in local part",
			input: "123@example.com",
			want:  "123@example.com",
		},
		{
			name:  "subdomain",
			input: "user@mail.example.com",
			want:  "user@mail.example.com",
		},
		{
			name:  "multiple subdomains",
			input: "user@mail.server.example.com",
			want:  "user@mail.server.example.com",
		},

		// Error cases
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "missing @ symbol",
			input:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "multiple @ symbols",
			input:   "user@@example.com",
			wantErr: true,
		},
		{
			name:    "@ at start",
			input:   "@example.com",
			wantErr: true,
		},
		{
			name:    "@ at end",
			input:   "user@",
			wantErr: true,
		},
		{
			name:    "no local part",
			input:   "@example.com",
			wantErr: true,
		},
		{
			name:    "no domain",
			input:   "user@",
			wantErr: true,
		},

		// Invalid local part characters
		{
			name:    "space in local part",
			input:   "user name@example.com",
			wantErr: true,
		},
		{
			name:    "comma in local part",
			input:   "user,name@example.com",
			wantErr: true,
		},
		{
			name:    "semicolon in local part",
			input:   "user;name@example.com",
			wantErr: true,
		},
		{
			name:    "colon in local part",
			input:   "user:name@example.com",
			wantErr: true,
		},
		{
			name:    "brackets in local part",
			input:   "user[name]@example.com",
			wantErr: true,
		},
		{
			name:    "parentheses in local part",
			input:   "user(name)@example.com",
			wantErr: true,
		},
		{
			name:    "unicode in local part",
			input:   "üser@example.com",
			wantErr: true,
		},
		{
			name:    "emoji in local part",
			input:   "user😀@example.com",
			wantErr: true,
		},

		// Invalid dots in local part
		{
			name:    "leading dot in local part",
			input:   ".user@example.com",
			wantErr: true,
		},
		{
			name:    "trailing dot in local part",
			input:   "user.@example.com",
			wantErr: true,
		},
		{
			name:    "double dot in local part",
			input:   "user..name@example.com",
			wantErr: true,
		},

		// Invalid domain
		{
			name:    "domain without dot",
			input:   "user@localhost",
			wantErr: true,
		},
		{
			name:    "domain with leading dot",
			input:   "user@.example.com",
			wantErr: true,
		},
		{
			name:    "domain with trailing dot",
			input:   "user@example.com.",
			wantErr: true,
		},
		{
			name:    "domain with double dot",
			input:   "user@example..com",
			wantErr: true,
		},
		{
			name:    "invalid IDN domain",
			input:   "user@-example.com",
			wantErr: true,
		},

		// Plus addressing edge cases
		{
			name:    "only plus sign in local part after normalization",
			input:   "+tag@gmail.com",
			wantErr: true,
		},
		{
			name:    "gmail dots removal results in empty local part",
			input:   ".+tag@gmail.com",
			wantErr: true,
		},

		// Complex valid cases
		{
			name:  "complex valid email with all features",
			input: `"Test User" <test.user+tag@gmail.com>`,
			want:  "testuser@gmail.com",
		},
		{
			name:  "mixed case with special chars",
			input: "UsEr.NaMe_123-test%@ExAmPle.CoM",
			want:  "user.name_123-test%@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err != ErrInvalidEmail {
				t.Errorf("Normalize() error = %v, want %v", err, ErrInvalidEmail)
				return
			}
			if got != tt.want {
				t.Errorf("Normalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateLocalCharset(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid cases
		{
			name:  "lowercase letters",
			input: "abcdefghijklmnopqrstuvwxyz",
			want:  true,
		},
		{
			name:  "numbers",
			input: "0123456789",
			want:  true,
		},
		{
			name:  "allowed special characters",
			input: "._%-+",
			want:  true,
		},
		{
			name:  "mixed valid characters",
			input: "user.name_123-test%+tag",
			want:  true,
		},
		{
			name:  "empty string",
			input: "",
			want:  true,
		},

		// Invalid cases
		{
			name:  "uppercase letters",
			input: "ABC",
			want:  false,
		},
		{
			name:  "space",
			input: "user name",
			want:  false,
		},
		{
			name:  "comma",
			input: "user,name",
			want:  false,
		},
		{
			name:  "at symbol",
			input: "user@name",
			want:  false,
		},
		{
			name:  "unicode character",
			input: "üser",
			want:  false,
		},
		{
			name:  "emoji",
			input: "user😀",
			want:  false,
		},
		{
			name:  "non-ascii",
			input: "user\x80",
			want:  false,
		},
		{
			name:  "parentheses",
			input: "user(name)",
			want:  false,
		},
		{
			name:  "brackets",
			input: "user[name]",
			want:  false,
		},
		{
			name:  "exclamation",
			input: "user!",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateLocalCharset(tt.input); got != tt.want {
				t.Errorf("validateLocalCharset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasLeadingOrTrailingDotOrDoubleDot(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Cases that should return true
		{
			name:  "leading dot",
			input: ".user",
			want:  true,
		},
		{
			name:  "trailing dot",
			input: "user.",
			want:  true,
		},
		{
			name:  "double dot",
			input: "user..name",
			want:  true,
		},
		{
			name:  "multiple double dots",
			input: "user..name..test",
			want:  true,
		},
		{
			name:  "only dots",
			input: "..",
			want:  true,
		},
		{
			name:  "single dot",
			input: ".",
			want:  true,
		},

		// Cases that should return false
		{
			name:  "no dots",
			input: "username",
			want:  false,
		},
		{
			name:  "single dot in middle",
			input: "user.name",
			want:  false,
		},
		{
			name:  "multiple single dots",
			input: "user.name.test",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasLeadingOrTrailingDotOrDoubleDot(tt.input); got != tt.want {
				t.Errorf("hasLeadingOrTrailingDotOrDoubleDot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizedType(t *testing.T) {
	// Test that Normalized type can be used as a string
	n := Normalized("test@example.com")

	// Test string conversion
	s := string(n)
	if s != "test@example.com" {
		t.Errorf("string(Normalized) = %v, want %v", s, "test@example.com")
	}

	// Test that empty Normalized is valid
	var empty Normalized
	if string(empty) != "" {
		t.Errorf("empty Normalized should be empty string, got %v", empty)
	}
}

// Benchmark tests
func BenchmarkNormalizeSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Normalize("user@example.com")
	}
}

func BenchmarkNormalizeGmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Normalize("john.doe+tag@gmail.com")
	}
}

func BenchmarkNormalizeIDN(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Normalize("user@münchen.de")
	}
}

func BenchmarkNormalizeComplex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Normalize(`"John Doe" <john.doe+tag@gmail.com>`)
	}
}
