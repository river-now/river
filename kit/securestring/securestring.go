// This package assumes that the caller's use case is not sensitive to
// timing attacks. In other words, it assumes that, even if an attacker
// can figure out the current index of the secret key originally used
// to encrypt the data, that information would not be materially useful
// to them. This is a reasonable assumption for most use cases.
package securestring

import (
	"fmt"
	"strconv"

	"github.com/river-now/river/kit/bytesutil"
	"github.com/river-now/river/kit/cryptoutil"
	"github.com/river-now/river/kit/keyset"
)

const current_pkg_version byte = 1

const one_mb_in_bytes = 1 << 20
const one_and_one_third_mb_in_bytes = one_mb_in_bytes + one_mb_in_bytes/3

type SecureString string // Base64-encoded, encrypted value
type RawValue any        // Any pre-serialization value

func Serialize(ks *keyset.Keyset, rv RawValue) (SecureString, error) {
	if rv == nil {
		return "", fmt.Errorf("invalid raw value: nil value")
	}
	if err := ks.Validate(); err != nil {
		return "", fmt.Errorf("invalid keyset: %w", err)
	}
	gob_value, err := bytesutil.ToGob(rv)
	if err != nil {
		return "", fmt.Errorf("error encoding value to gob: %w", err)
	}
	plaintext := append([]byte{current_pkg_version}, gob_value...)
	firstKey, err := ks.First()
	if err != nil {
		return "", fmt.Errorf("error getting first key from keyset: %w", err)
	}
	ciphertext, err := cryptoutil.EncryptSymmetricXChaCha20Poly1305(plaintext, firstKey)
	if err != nil {
		return "", fmt.Errorf("error encrypting value: %w", err)
	}
	if len(ciphertext) > one_mb_in_bytes {
		return "", fmt.Errorf("ciphertext too large (over 1MB)")
	}
	return SecureString(bytesutil.ToBase64(ciphertext)), nil
}

func Deserialize[T any](ks *keyset.Keyset, ss SecureString) (T, error) {
	var zeroT T
	if len(ss) == 0 {
		return zeroT, fmt.Errorf("invalid secure string: empty value")
	}
	if len(ss) > one_and_one_third_mb_in_bytes {
		return zeroT, fmt.Errorf("secure string too large (over 1.33MB)")
	}
	if err := ks.Validate(); err != nil {
		return zeroT, fmt.Errorf("invalid keyset: %w", err)
	}
	ciphertext, err := bytesutil.FromBase64(string(ss))
	if err != nil {
		return zeroT, fmt.Errorf("error decoding base64: %w", err)
	}
	if len(ciphertext) > one_mb_in_bytes {
		return zeroT, fmt.Errorf("ciphertext too large (over 1MB)")
	}
	plaintext, err := keyset.Attempt(ks, func(k cryptoutil.Key32) ([]byte, error) {
		return cryptoutil.DecryptSymmetricXChaCha20Poly1305(ciphertext, k)
	})
	if err != nil {
		return zeroT, fmt.Errorf("error decrypting value: %w", err)
	}
	version := plaintext[0]
	if version != current_pkg_version {
		return zeroT, fmt.Errorf("unsupported SecureString version %d", version)
	}
	var out T
	if err := bytesutil.FromGobInto(plaintext[1:], &out); err != nil {
		return zeroT, fmt.Errorf("error decoding gob: %w", err)
	}
	return out, nil
}

// Deprecated: Use only if you need to support legacy encrypted values.
func GetLegacyHKDFInfoStr() string {
	return "river_kit_securestring_v" +
		strconv.Itoa(int(current_pkg_version)) +
		"_encryption_key"
}
