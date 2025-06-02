package keyset

// __TODO add test suite

import (
	"fmt"
	"os"
	"sync"

	"github.com/river-now/river/kit/bytesutil"
	"github.com/river-now/river/kit/cryptoutil"
	"github.com/river-now/river/kit/lazyget"
)

// Base64-encoded 32-byte root secret.
// You can generate new root secrets using the following command:
// `openssl rand -base64 32`.
type RootSecret string

// Latest-first slice of base64-encoded 32-byte root secrets.
// You can generate new root secrets using the following command:
// `openssl rand -base64 32`.
type RootSecrets []RootSecret

// Latest-first slice of size 32 byte array pointers
type Keyset []cryptoutil.Key32

type ApplicationKeyset struct {
	// Provide a latest-first slice of environment variable names pointing
	// to base64-encoded 32-byte root secrets.
	// Example: MustLoadRootKeyset("CURRENT_SECRET", "PREVIOUS_SECRET")
	LatestFirstEnvVarNames []string
	// Used as the HKDF salt param when creating scoped keysets.
	ApplicationName string
	once            sync.Once
	rootKeyset      Keyset
}

// Returns a getter function that returns a Keyset derived from the root
// keyset using HKDF with the instance's ApplicationName and the provided
// purpose string. The ApplicationName is used as the HKDF salt param, and
// the purpose string is used as the HKDF info param. Panics if anything
// is misconfigured.
func (k *ApplicationKeyset) ScopedGetterMust(purpose string) func() Keyset {
	if len(k.LatestFirstEnvVarNames) == 0 {
		panic("ApplicationKeyset.LatestFirstEnvVarNames must not be empty")
	}
	if k.ApplicationName == "" {
		panic("ApplicationKeyset.ApplicationName must not be empty")
	}
	k.once.Do(func() {
		k.rootKeyset = MustLoadRootKeyset(k.LatestFirstEnvVarNames...)
	})
	return lazyget.New(func() Keyset {
		return k.rootKeyset.MustHKDF([]byte(k.ApplicationName), purpose)
	})
}

// Attempt runs the provided function for each key in the keyset
// until either (i) an attempt does not return an error (meaning
// it succeeded) or (ii) all keys have been attempted. This is
// useful when you want to fallback to a prior key if the current
// key fails due to a recent rotation.
func Attempt[R any](keyset Keyset, f func(cryptoutil.Key32) (R, error)) (R, error) {
	if len(keyset) == 0 {
		return *new(R), fmt.Errorf("keyset is empty")
	}
	var lastErr error
	for _, k := range keyset {
		result, err := f(k)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return *new(R), lastErr
}

// Keyset.MustHKDF applies HKDF to each key in the base Keyset using
// the provided salt and info string, returning a new Keyset consisting
// of the derived keys, and panics if an error occurs.
func (ks Keyset) MustHKDF(salt []byte, info string) Keyset {
	derivedKeys, err := ks.HKDF(salt, info)
	if err != nil {
		panic(fmt.Sprintf("error deriving keys from keyset: %v", err))
	}
	return derivedKeys
}

// Keyset.HKDF applies HKDF to each key in the base Keyset using the
// provided salt and info string, returning a new Keyset consisting
// of the derived keys.
func (ks Keyset) HKDF(salt []byte, info string) (Keyset, error) {
	if len(ks) == 0 {
		return nil, fmt.Errorf("root keyset is empty")
	}
	derivedKeys := make(Keyset, 0, len(ks))
	for i, rootKey := range ks {
		dk, err := cryptoutil.HkdfSha256(rootKey, salt, info)
		if err != nil {
			return nil, fmt.Errorf("error deriving key from root key %d: %w", i, err)
		}
		derivedKeys = append(derivedKeys, dk)
	}
	return derivedKeys, nil
}

// Pass in a latest-first slice of environment variable names pointing
// to base64-encoded 32-byte root secrets.
// Example: MustLoadRootKeyset("CURRENT_SECRET", "PREVIOUS_SECRET")
func MustLoadRootKeyset(envVarNames ...string) Keyset {
	keyset, err := LoadRootKeyset(envVarNames...)
	if err != nil {
		panic(fmt.Sprintf("error loading root keyset: %v", err))
	}
	return keyset
}

// Pass in a latest-first slice of environment variable names pointing
// to base64-encoded 32-byte root secrets.
// Example: LoadRootKeyset("CURRENT_SECRET", "PREVIOUS_SECRET")
func LoadRootKeyset(envVarNames ...string) (Keyset, error) {
	rootSecrets, err := LoadRootSecrets(envVarNames...)
	if err != nil {
		return nil, fmt.Errorf("error loading root secrets: %w", err)
	}
	keyset, err := RootSecretsToRootKeyset(rootSecrets)
	if err != nil {
		return nil, fmt.Errorf("error converting root secrets to keyset: %w", err)
	}
	return keyset, nil
}

// RootSecretsToRootKeyset converts a slice of base64-encoded root
// secrets into a Keyset.
func RootSecretsToRootKeyset(rootSecrets RootSecrets) (Keyset, error) {
	if len(rootSecrets) == 0 {
		return nil, fmt.Errorf("at least 1 root secret is required")
	}
	keys := make(Keyset, 0, len(rootSecrets))
	for i, secret := range rootSecrets {
		secretBytes, err := bytesutil.FromBase64(string(secret))
		if err != nil {
			return nil, fmt.Errorf(
				"error decoding base64 secret %d: %w", i, err,
			)
		}
		if len(secretBytes) != cryptoutil.KeySize {
			return nil, fmt.Errorf("secret %d is not 32 bytes", i)
		}
		keys = append(keys, cryptoutil.Key32(secretBytes))
	}
	return keys, nil
}

// Pass in a latest-first slice of environment variable names pointing
// to base64-encoded 32-byte root secrets.
// Example: LoadRootSecrets("CURRENT_SECRET", "PREVIOUS_SECRET")
func LoadRootSecrets(envVarNames ...string) (RootSecrets, error) {
	if len(envVarNames) == 0 {
		return nil, fmt.Errorf("at least 1 env var key is required")
	}
	rootSecrets := make(RootSecrets, 0, len(envVarNames))
	for _, envVarKey := range envVarNames {
		if envVarKey == "" {
			return nil, fmt.Errorf("env var key %s is empty", envVarKey)
		}
		secret := os.Getenv(envVarKey)
		if secret == "" {
			return nil, fmt.Errorf("env var %s is not set", envVarKey)
		}
		rootSecrets = append(rootSecrets, RootSecret(secret))
	}
	return rootSecrets, nil
}
