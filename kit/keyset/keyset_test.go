package keyset

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/river-now/river/kit/cryptoutil"
)

// Helper function to create a valid base64-encoded 32-byte secret
func generateTestSecret() string {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	return base64.StdEncoding.EncodeToString(secret)
}

// Helper function to create a test Key32
func generateTestKey32() cryptoutil.Key32 {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 100)
	}
	k32, _ := cryptoutil.ToKey32(key)
	return k32
}

func TestKeyset_Unwrap(t *testing.T) {
	keys := UnwrappedKeyset{generateTestKey32(), generateTestKey32()}
	ks := &Keyset{UnwrappedKeyset: keys}

	unwrapped := ks.Unwrap()
	if len(unwrapped) != 2 {
		t.Errorf("expected 2 keys, got %d", len(unwrapped))
	}

	// Verify it returns the same reference
	if &unwrapped[0] != &keys[0] {
		t.Error("Unwrap should return the same reference")
	}
}

func TestKeyset_First(t *testing.T) {
	tests := []struct {
		name    string
		keyset  *Keyset
		wantErr bool
	}{
		{
			name:    "empty keyset",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{}},
			wantErr: true,
		},
		{
			name:    "nil first key",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{nil}},
			wantErr: true,
		},
		{
			name:    "valid first key",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{generateTestKey32()}},
			wantErr: false,
		},
		{
			name:    "multiple keys",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{generateTestKey32(), generateTestKey32()}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := tt.keyset.First()
			if (err != nil) != tt.wantErr {
				t.Errorf("First() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && key == nil {
				t.Error("expected non-nil key")
			}
		})
	}
}

func TestAttempt(t *testing.T) {
	successKey := generateTestKey32()
	failKey := generateTestKey32()

	tests := []struct {
		name      string
		keyset    *Keyset
		fn        func(cryptoutil.Key32) (string, error)
		wantValue string
		wantErr   bool
	}{
		{
			name:    "empty keyset",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{}},
			fn:      func(k cryptoutil.Key32) (string, error) { return "ok", nil },
			wantErr: true,
		},
		{
			name:    "nil key in keyset",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{nil}},
			fn:      func(k cryptoutil.Key32) (string, error) { return "ok", nil },
			wantErr: true,
		},
		{
			name:   "first key succeeds",
			keyset: &Keyset{UnwrappedKeyset: UnwrappedKeyset{successKey, failKey}},
			fn: func(k cryptoutil.Key32) (string, error) {
				if bytes.Equal(k[:], successKey[:]) {
					return "success", nil
				}
				return "", errors.New("wrong key")
			},
			wantValue: "success",
			wantErr:   false,
		},
		{
			name:   "fallback to second key",
			keyset: &Keyset{UnwrappedKeyset: UnwrappedKeyset{failKey, successKey}},
			fn: func(k cryptoutil.Key32) (string, error) {
				if bytes.Equal(k[:], successKey[:]) {
					return "success", nil
				}
				return "", errors.New("wrong key")
			},
			wantValue: "success",
			wantErr:   false,
		},
		{
			name:   "all keys fail",
			keyset: &Keyset{UnwrappedKeyset: UnwrappedKeyset{failKey, failKey}},
			fn: func(k cryptoutil.Key32) (string, error) {
				return "", errors.New("always fail")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Attempt(tt.keyset, tt.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("Attempt() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.wantValue {
				t.Errorf("Attempt() result = %v, want %v", result, tt.wantValue)
			}
		})
	}
}

func TestKeyset_HKDF(t *testing.T) {
	tests := []struct {
		name    string
		keyset  *Keyset
		salt    []byte
		info    string
		wantErr bool
	}{
		{
			name:    "empty keyset",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{}},
			salt:    []byte("salt"),
			info:    "info",
			wantErr: true,
		},
		{
			name:    "single key",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{generateTestKey32()}},
			salt:    []byte("salt"),
			info:    "info",
			wantErr: false,
		},
		{
			name:    "multiple keys",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{generateTestKey32(), generateTestKey32()}},
			salt:    []byte("salt"),
			info:    "info",
			wantErr: false,
		},
		{
			name:    "empty salt",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{generateTestKey32()}},
			salt:    []byte{},
			info:    "info",
			wantErr: false,
		},
		{
			name:    "empty info",
			keyset:  &Keyset{UnwrappedKeyset: UnwrappedKeyset{generateTestKey32()}},
			salt:    []byte("salt"),
			info:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			derived, err := tt.keyset.HKDF(tt.salt, tt.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("HKDF() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if derived == nil {
					t.Error("expected non-nil derived keyset")
				} else if len(derived.UnwrappedKeyset) != len(tt.keyset.UnwrappedKeyset) {
					t.Errorf("expected %d derived keys, got %d",
						len(tt.keyset.UnwrappedKeyset), len(derived.UnwrappedKeyset))
				}
			}
		})
	}
}

func TestRootSecretsToRootKeyset(t *testing.T) {
	validSecret := generateTestSecret()
	invalidBase64 := "not-valid-base64!"
	wrongSizeSecret := base64.StdEncoding.EncodeToString([]byte("too short"))

	tests := []struct {
		name    string
		secrets RootSecrets
		wantErr bool
	}{
		{
			name:    "empty secrets",
			secrets: RootSecrets{},
			wantErr: true,
		},
		{
			name:    "single valid secret",
			secrets: RootSecrets{RootSecret(validSecret)},
			wantErr: false,
		},
		{
			name:    "multiple valid secrets",
			secrets: RootSecrets{RootSecret(validSecret), RootSecret(generateTestSecret())},
			wantErr: false,
		},
		{
			name:    "invalid base64",
			secrets: RootSecrets{RootSecret(invalidBase64)},
			wantErr: true,
		},
		{
			name:    "wrong size secret",
			secrets: RootSecrets{RootSecret(wrongSizeSecret)},
			wantErr: true,
		},
		{
			name:    "mix of valid and invalid",
			secrets: RootSecrets{RootSecret(validSecret), RootSecret(invalidBase64)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyset, err := RootSecretsToRootKeyset(tt.secrets)
			if (err != nil) != tt.wantErr {
				t.Errorf("RootSecretsToRootKeyset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && keyset == nil {
				t.Error("expected non-nil keyset")
			}
			if !tt.wantErr && len(keyset.UnwrappedKeyset) != len(tt.secrets) {
				t.Errorf("expected %d keys, got %d", len(tt.secrets), len(keyset.UnwrappedKeyset))
			}
		})
	}
}

func TestLoadRootSecrets(t *testing.T) {
	// Setup test environment variables
	os.Setenv("TEST_SECRET_1", generateTestSecret())
	os.Setenv("TEST_SECRET_2", generateTestSecret())
	defer os.Unsetenv("TEST_SECRET_1")
	defer os.Unsetenv("TEST_SECRET_2")

	tests := []struct {
		name    string
		envVars []string
		wantErr bool
		wantLen int
	}{
		{
			name:    "no env vars",
			envVars: []string{},
			wantErr: true,
		},
		{
			name:    "single valid env var",
			envVars: []string{"TEST_SECRET_1"},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "multiple valid env vars",
			envVars: []string{"TEST_SECRET_1", "TEST_SECRET_2"},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "empty env var name",
			envVars: []string{""},
			wantErr: true,
		},
		{
			name:    "non-existent env var",
			envVars: []string{"DOES_NOT_EXIST"},
			wantErr: true,
		},
		{
			name:    "mix of valid and invalid",
			envVars: []string{"TEST_SECRET_1", "DOES_NOT_EXIST"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := LoadRootSecrets(tt.envVars...)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRootSecrets() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(secrets) != tt.wantLen {
				t.Errorf("expected %d secrets, got %d", tt.wantLen, len(secrets))
			}
		})
	}
}

func TestLoadRootKeyset(t *testing.T) {
	// Setup test environment variables
	os.Setenv("TEST_KEYSET_1", generateTestSecret())
	os.Setenv("TEST_KEYSET_2", generateTestSecret())
	os.Setenv("TEST_INVALID", "invalid-base64!")
	defer os.Unsetenv("TEST_KEYSET_1")
	defer os.Unsetenv("TEST_KEYSET_2")
	defer os.Unsetenv("TEST_INVALID")

	tests := []struct {
		name    string
		envVars []string
		wantErr bool
	}{
		{
			name:    "valid single key",
			envVars: []string{"TEST_KEYSET_1"},
			wantErr: false,
		},
		{
			name:    "valid multiple keys",
			envVars: []string{"TEST_KEYSET_1", "TEST_KEYSET_2"},
			wantErr: false,
		},
		{
			name:    "invalid secret format",
			envVars: []string{"TEST_INVALID"},
			wantErr: true,
		},
		{
			name:    "non-existent env var",
			envVars: []string{"DOES_NOT_EXIST"},
			wantErr: true,
		},
		{
			name:    "no env vars",
			envVars: []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyset, err := LoadRootKeyset(tt.envVars...)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadRootKeyset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && keyset == nil {
				t.Error("expected non-nil keyset")
			}
		})
	}
}

func TestToAppKeyset(t *testing.T) {
	// Setup test environment variable
	os.Setenv("TEST_APP_SECRET", generateTestSecret())
	defer os.Unsetenv("TEST_APP_SECRET")

	// Test panic cases
	t.Run("panic on empty env vars", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty env vars")
			}
		}()
		ToAppKeyset(AppKeysetConfig{
			LatestFirstEnvVarNames: []string{},
			ApplicationName:        "test-app",
		})
	})

	t.Run("panic on empty application name", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty application name")
			}
		}()
		ToAppKeyset(AppKeysetConfig{
			LatestFirstEnvVarNames: []string{"TEST_APP_SECRET"},
			ApplicationName:        "",
		})
	})

	// Test successful creation
	t.Run("valid config", func(t *testing.T) {
		appKeyset := ToAppKeyset(AppKeysetConfig{
			LatestFirstEnvVarNames: []string{"TEST_APP_SECRET"},
			ApplicationName:        "test-app",
		})

		if appKeyset == nil {
			t.Fatal("expected non-nil AppKeyset")
		}

		// Test Root() function
		root := appKeyset.Root()
		if root == nil {
			t.Error("expected non-nil root keyset")
		}

		// Test HKDF() function
		hkdfFn := appKeyset.HKDF("test-purpose")
		if hkdfFn == nil {
			t.Error("expected non-nil HKDF function")
		}

		derived := hkdfFn()
		if derived == nil {
			t.Error("expected non-nil derived keyset")
		}

		// Verify same result when called again (lazy loading)
		derived2 := hkdfFn()
		if derived != derived2 {
			t.Error("expected same keyset instance from lazy loading")
		}
	})

	// Test panic on empty HKDF purpose
	t.Run("panic on empty HKDF purpose", func(t *testing.T) {
		appKeyset := ToAppKeyset(AppKeysetConfig{
			LatestFirstEnvVarNames: []string{"TEST_APP_SECRET"},
			ApplicationName:        "test-app",
		})

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty HKDF purpose")
			}
		}()

		hkdfFn := appKeyset.HKDF("")
		hkdfFn() // Should panic here
	})
}

func TestAppKeyset_MultipleHKDFPurposes(t *testing.T) {
	os.Setenv("TEST_MULTI_SECRET", generateTestSecret())
	defer os.Unsetenv("TEST_MULTI_SECRET")

	appKeyset := ToAppKeyset(AppKeysetConfig{
		LatestFirstEnvVarNames: []string{"TEST_MULTI_SECRET"},
		ApplicationName:        "test-app",
	})

	// Get different derived keysets for different purposes
	encryptionKeys := appKeyset.HKDF("encryption")()
	signingKeys := appKeyset.HKDF("signing")()

	if encryptionKeys == nil || signingKeys == nil {
		t.Fatal("expected non-nil derived keysets")
	}

	// Verify they're different keysets
	if encryptionKeys == signingKeys {
		t.Error("expected different keyset instances for different purposes")
	}

	// Verify the keys themselves are different
	encKey, _ := encryptionKeys.First()
	sigKey, _ := signingKeys.First()

	if bytes.Equal(encKey[:], sigKey[:]) {
		t.Error("expected different keys for different purposes")
	}
}
