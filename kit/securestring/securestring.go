// This package assumes that the caller's use case is not sensitive to
// timing attacks. In other words, it assumes that, even if an attacker
// can figure out the current index of the secret key originally used
// to encrypt the data, that information would not be materially useful
// to them. This is a reasonable assumption for most use cases.
package securestring

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/river-now/river/kit/bytesutil"
	"github.com/river-now/river/kit/cryptoutil"
	"golang.org/x/crypto/hkdf"
)

const current_pkg_version byte = 1

const key_size = 32 // The size, in bytes, of secrets and keys used by this package

const one_mb_in_bytes = 1 << 20
const one_and_one_third_mb_in_bytes = one_mb_in_bytes + one_mb_in_bytes/3

type Secret string               // 32-byte, base64-encoded secret
type LatestFirstSecrets []Secret // Latest-first slice of 32-byte, base64-encoded secrets
type Salt []byte                 // Used to add domain separation to derived keys. Does not need to be secret or random.
type Key *[key_size]byte         // 32-byte key for encryption
type SecureString string         // Base64-encoded, encrypted value
type RawValue any                // Any pre-serialization value

func Serialize(keys []Key, rv RawValue) (SecureString, error) {
	if len(keys) == 0 {
		return "", fmt.Errorf("invalid keys container: at least one key is required")
	}
	if rv == nil {
		return "", fmt.Errorf("invalid raw value: nil value")
	}
	gob_value, err := bytesutil.ToGob(rv)
	if err != nil {
		return "", fmt.Errorf("error encoding value to gob: %w", err)
	}
	plaintext := append([]byte{current_pkg_version}, gob_value...)
	if keys[0] == nil {
		return "", fmt.Errorf("invalid current key: nil value")
	}
	ciphertext, err := cryptoutil.EncryptSymmetricXChaCha20Poly1305(plaintext, keys[0])
	if err != nil {
		return "", fmt.Errorf("error encrypting value: %w", err)
	}
	if len(ciphertext) > one_mb_in_bytes {
		return "", fmt.Errorf("ciphertext too large (over 1MB)")
	}
	return SecureString(bytesutil.ToBase64(ciphertext)), nil
}

func Deserialize[T any](keys []Key, ss SecureString) (T, error) {
	if len(keys) == 0 {
		return *new(T), fmt.Errorf("invalid keys container: at least one key is required")
	}
	if len(ss) == 0 {
		return *new(T), fmt.Errorf("invalid secure string: empty value")
	}
	if len(ss) > one_and_one_third_mb_in_bytes {
		return *new(T), fmt.Errorf("secure string too large (over 1.33MB)")
	}
	ciphertext, err := bytesutil.FromBase64(string(ss))
	if err != nil {
		return *new(T), fmt.Errorf("error decoding base64: %w", err)
	}
	if len(ciphertext) > one_mb_in_bytes {
		return *new(T), fmt.Errorf("ciphertext too large (over 1MB)")
	}
	var plaintext []byte
	var success_key Key
	for _, key := range keys {
		if key == nil {
			continue
		}
		p, err := cryptoutil.DecryptSymmetricXChaCha20Poly1305(ciphertext, key)
		if err == nil {
			plaintext = p
			success_key = key
			break
		}
	}
	if success_key == nil {
		return *new(T), errors.New("could not decrypt value with any key")
	}
	version := plaintext[0]
	if version != current_pkg_version {
		return *new(T), fmt.Errorf("unsupported SecureString version %d", version)
	}
	var out T
	if err := bytesutil.FromGobInto(plaintext[1:], &out); err != nil {
		return *new(T), fmt.Errorf("error decoding gob: %w", err)
	}
	return out, nil
}

func ParseSecrets(secrets LatestFirstSecrets, salt Salt) ([]Key, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("at least 1 secret is required (you can generate one by running `openssl rand -base64 32`)")
	}
	secrets_bytes := make([][key_size]byte, len(secrets))
	for i, secret := range secrets {
		bytes, err := bytesutil.FromBase64(string(secret))
		if err != nil {
			return nil, fmt.Errorf("error decoding base64: %w", err)
		}
		if len(bytes) != key_size {
			return nil, fmt.Errorf("secret %d is not 32 bytes", i)
		}
		copy(secrets_bytes[i][:], bytes)
	}
	kcs := make([]Key, len(secrets))
	for i := range secrets_bytes {
		kc, err := derive_encryption_key(&secrets_bytes[i], salt)
		if err != nil {
			return nil, fmt.Errorf("error deriving keys: %w", err)
		}
		kcs[i] = kc
	}
	return kcs, nil
}

func derive_encryption_key(master_key *[key_size]byte, salt Salt) (Key, error) {
	info := []byte("river_kit_securestring_v" + strconv.Itoa(int(current_pkg_version)) + "_encryption_key")
	var key [key_size]byte
	reader := hkdf.New(sha256.New, master_key[:], salt, info)
	if _, err := io.ReadFull(reader, key[:]); err != nil {
		return nil, fmt.Errorf("error deriving encryption key: %w", err)
	}
	return &key, nil
}
