package securestring

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/river-now/river/kit/bytesutil"
	"github.com/river-now/river/kit/cryptoutil"
	"github.com/river-now/river/kit/keyset"
	"golang.org/x/crypto/chacha20poly1305"
)

func randSecrets(n int) keyset.RootSecrets {
	out := make([]keyset.RootSecret, n)
	for i := range out {
		var b [cryptoutil.KeySize]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
		}
		out[i] = keyset.RootSecret(base64.StdEncoding.EncodeToString(b[:]))
	}
	return out
}

func mustKeys(t *testing.T, n int) []cryptoutil.Key32 {
	secrets, err := ParseSecrets(randSecrets(n), []byte("george_testing_salt"))
	if err != nil {
		t.Fatalf("ParseSecrets error: %v", err)
	}
	return secrets
}

func roundTrip[T comparable](t *testing.T, value T) {
	kcs := mustKeys(t, 1)
	ss, err := Serialize(kcs, value)
	if err != nil {
		t.Fatalf("Serialize failed for value %v: %v", value, err)
	}
	got, err := Deserialize[T](kcs, ss)
	if err != nil {
		t.Fatalf("Deserialize failed for value %v: %v", value, err)
	}
	if got != value {
		t.Fatalf("round‑trip mismatch: want %v, got %v", value, got)
	}
}

func TestSecureString_RoundTrip(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		roundTrip(t, "hello world")
	})

	t.Run("int", func(t *testing.T) {
		roundTrip(t, 42)
	})

	t.Run("struct", func(t *testing.T) {
		type demo struct {
			A int
			B string
		}
		roundTrip(t, demo{A: 7, B: "seven"})
	})
}

func TestSecureString_RoundTrip_PointerTypes(t *testing.T) {
	kcs := mustKeys(t, 1)

	type demoPtrStruct struct {
		Name  string
		Value int
	}

	t.Run("pointer to struct", func(t *testing.T) {
		original := &demoPtrStruct{Name: "TestPtr", Value: 123}
		ss, err := Serialize(kcs, original)
		if err != nil {
			t.Fatalf("Serialize failed for pointer to struct: %v", err)
		}

		got, err := Deserialize[*demoPtrStruct](kcs, ss)
		if err != nil {
			t.Fatalf("Deserialize failed for pointer to struct: %v", err)
		}

		if got == nil {
			t.Fatalf("Deserialize resulted in nil pointer, want non-nil")
		}
		if original.Name != got.Name || original.Value != got.Value {
			t.Fatalf("round-trip (pointer to struct) mismatch: want %+v, got %+v", *original, *got)
		}
	})

	t.Run("pointer to string", func(t *testing.T) {
		strValue := "hello pointer"
		original := &strValue

		ss, err := Serialize(kcs, original)
		if err != nil {
			t.Fatalf("Serialize failed for pointer to string: %v", err)
		}
		got, err := Deserialize[*string](kcs, ss)
		if err != nil {
			t.Fatalf("Deserialize failed for pointer to string: %v", err)
		}
		if got == nil {
			t.Fatalf("Deserialize resulted in nil string pointer")
		}
		if *original != *got {
			t.Fatalf("round-trip (pointer to string) mismatch: want %q, got %q", *original, *got)
		}
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		var typedNil *demoPtrStruct = nil
		_, err := Serialize(kcs, typedNil)
		if err == nil {
			t.Fatalf("Expected Serialize to fail for typed nil pointer, but it succeeded")
		}
	})
}

func TestSecureString_WrongKeyFails(t *testing.T) {
	good := mustKeys(t, 1)
	bad := mustKeys(t, 1)

	ss, err := Serialize(good, "secret data")
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}
	if _, err = Deserialize[string](bad, ss); err == nil {
		t.Fatalf("expected decryption failure with wrong key")
	}
}

func TestSecureString_SizeLimits(t *testing.T) {
	kcs := mustKeys(t, 1)

	t.Run("Serialize payload too large", func(t *testing.T) {
		big := make([]byte, one_mb_in_bytes+1)
		if _, err := Serialize(kcs, big); err == nil {
			t.Fatalf("expected Serialize to fail for payload >1 MiB")
		}
	})

	t.Run("Deserialize SecureString too large", func(t *testing.T) {
		baseSizeForOver1MBDecoded := one_mb_in_bytes - 100
		payloadJustUnderLimitEncoded := make([]byte, baseSizeForOver1MBDecoded/3*4+3)
		rand.Read(payloadJustUnderLimitEncoded)
		for i, char := range payloadJustUnderLimitEncoded {
			payloadJustUnderLimitEncoded[i] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"[char%64]
		}
		oversizeUnderlyingData := make([]byte, one_mb_in_bytes)
		ss := SecureString(base64.StdEncoding.EncodeToString(oversizeUnderlyingData) + "extraPaddingMakesItTooLong")
		if len(ss) <= one_and_one_third_mb_in_bytes {
			t.Logf("Generated SecureString length %d, limit %d. Adjusting to ensure it's over.", len(ss), one_and_one_third_mb_in_bytes)
			paddingNeeded := (one_and_one_third_mb_in_bytes - len(ss)) + 10
			if paddingNeeded <= 0 {
				paddingNeeded = 10
			}
			ss = SecureString(string(ss) + strings.Repeat("A", paddingNeeded))
		}

		if _, err := Deserialize[string](kcs, ss); err == nil {
			t.Fatalf("expected Deserialize to fail for oversized base64 input (len(ss) check)")
		}
	})
}

func TestParseSecrets_Errors(t *testing.T) {
	t.Run("empty secrets slice", func(t *testing.T) {
		if _, err := ParseSecrets(nil, []byte("bob")); err == nil {
			t.Fatalf("expected error for empty secrets slice")
		}
	})

	t.Run("invalid base64 secret", func(t *testing.T) {
		bad := keyset.RootSecrets{"$$not base64$$"}
		if _, err := ParseSecrets(bad, []byte("sally")); err == nil {
			t.Fatalf("expected error for invalid base64 secret")
		}
	})

	t.Run("secret wrong length after decode", func(t *testing.T) {
		tests := []struct {
			name        string
			secretBytes []byte
			expectedLen int
		}{
			{"too short", []byte("not32bytes"), len("not32bytes")},
			{"too long", []byte("this secret is definitely longer than 32 bytes"), len("this secret is definitely longer than 32 bytes")},
			{"exactly 31 bytes", make([]byte, 31), 31},
			{"exactly 33 bytes", make([]byte, 33), 33},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if strings.HasPrefix(tc.name, "exactly") {
					for k := range tc.secretBytes {
						tc.secretBytes[k] = byte(k + 1)
					}
				}
				secretB64 := base64.StdEncoding.EncodeToString(tc.secretBytes)
				secrets := keyset.RootSecrets{keyset.RootSecret(secretB64)}
				if _, err := ParseSecrets(secrets, []byte("test")); err == nil {
					t.Fatalf("expected error for secret not %d bytes long (was %d)", cryptoutil.KeySize, tc.expectedLen)
				}
			})
		}
	})
}

func TestParseSecrets_SaltVariations(t *testing.T) {
	goodSecrets := randSecrets(1)

	t.Run("nil salt", func(t *testing.T) {
		keys, err := ParseSecrets(goodSecrets, nil)
		if err != nil {
			t.Fatalf("ParseSecrets failed with nil salt: %v", err)
		}
		if len(keys) != 1 || keys[0] == nil {
			t.Fatalf("expected valid keys with nil salt, got keys: %v", keys)
		}
		ss, err := Serialize(keys, "test nil salt value")
		if err != nil {
			t.Fatalf("Serialize failed with key derived from nil salt: %v", err)
		}
		if _, err := Deserialize[string](keys, ss); err != nil {
			t.Fatalf("Deserialize failed with key derived from nil salt: %v", err)
		}
	})

	t.Run("empty salt", func(t *testing.T) {
		keys, err := ParseSecrets(goodSecrets, []byte(""))
		if err != nil {
			t.Fatalf("ParseSecrets failed with empty salt: %v", err)
		}
		if len(keys) != 1 || keys[0] == nil {
			t.Fatalf("expected valid keys with empty salt, got keys: %v", keys)
		}
		ss, err := Serialize(keys, "test empty salt value")
		if err != nil {
			t.Fatalf("Serialize failed with key derived from empty salt: %v", err)
		}
		if _, err := Deserialize[string](keys, ss); err != nil {
			t.Fatalf("Deserialize failed with key derived from empty salt: %v", err)
		}
	})

	t.Run("different salts same secret", func(t *testing.T) {
		salt1 := []byte("salt_one_pepper_unique")
		salt2 := []byte("salt_two_sugar_unique")

		keys1, err := ParseSecrets(goodSecrets, salt1)
		if err != nil {
			t.Fatalf("ParseSecrets error with salt1: %v", err)
		}
		keys2, err := ParseSecrets(goodSecrets, salt2)
		if err != nil {
			t.Fatalf("ParseSecrets error with salt2: %v", err)
		}

		if len(keys1) != 1 || keys1[0] == nil {
			t.Fatal("keys1 is invalid")
		}
		if len(keys2) != 1 || keys2[0] == nil {
			t.Fatal("keys2 is invalid")
		}

		if reflect.DeepEqual(keys1[0], keys2[0]) {
			t.Fatalf("Expected different derived keys for different salts, but they were the same.")
		}

		originalValue := "test salt difference"
		ss1, err := Serialize(keys1, originalValue)
		if err != nil {
			t.Fatalf("Serialize with keys1 failed: %v", err)
		}

		_, err = Deserialize[string](keys2, ss1)
		if err == nil {
			t.Fatalf("expected decryption to fail when salt differs but secret is the same")
		}

		got, err := Deserialize[string](keys1, ss1)
		if err != nil {
			t.Fatalf("Deserialize with original keys1 failed: %v", err)
		}
		if got != originalValue {
			t.Fatalf("Deserialize mismatch with original keys1: want %q, got %q", originalValue, got)
		}
	})
}

func TestSecureString_KeyRotation(t *testing.T) {
	oldKeyContainer := mustKeys(t, 1)
	newKeyContainer := mustKeys(t, 1)

	if reflect.DeepEqual(oldKeyContainer[0], newKeyContainer[0]) {
		t.Fatalf("Test setup error: oldKey and newKey are the same, ensure mustKeys generates unique secrets.")
	}

	rotatedKeys := []cryptoutil.Key32{newKeyContainer[0], oldKeyContainer[0]}

	value := "sensitive data for rotation"
	ss, err := Serialize(oldKeyContainer, value)
	if err != nil {
		t.Fatalf("Serialize with oldKey failed: %v", err)
	}

	got, err := Deserialize[string](rotatedKeys, ss)
	if err != nil {
		t.Fatalf("Deserialize with rotated keys failed: %v", err)
	}
	if got != value {
		t.Fatalf("rotation mismatch: want %q, got %q", value, got)
	}

	valueNew := "new sensitive data"
	ssNew, err := Serialize(newKeyContainer, valueNew)
	if err != nil {
		t.Fatalf("Serialize with newKey failed: %v", err)
	}
	oldFirstRotatedKeys := []cryptoutil.Key32{oldKeyContainer[0], newKeyContainer[0]}
	gotNew, err := Deserialize[string](oldFirstRotatedKeys, ssNew)
	if err != nil {
		t.Fatalf("Deserialize with oldFirstRotatedKeys failed: %v", err)
	}
	if gotNew != valueNew {
		t.Fatalf("rotation mismatch for new key: want %q, got %q", valueNew, gotNew)
	}
}

func TestSecureString_EmptyInput(t *testing.T) {
	kcs := mustKeys(t, 1)

	t.Run("empty string", func(t *testing.T) {
		roundTrip(t, "")
	})

	t.Run("empty struct", func(t *testing.T) {
		type empty struct{}
		roundTrip(t, empty{})
	})

	t.Run("nil value (any(nil))", func(t *testing.T) {
		if _, err := Serialize(kcs, nil); err == nil {
			t.Fatalf("expected error when serializing nil (interface{}(nil))")
		}
	})
}

func TestSecureString_InvalidInputs(t *testing.T) {
	kcsValid := mustKeys(t, 1)
	validValue := "some test data for invalid inputs"
	ssValid, errSerialize := Serialize(kcsValid, validValue)
	if errSerialize != nil {
		t.Fatalf("Setup: Serialize for TestSecureString_InvalidInputs failed: %v", errSerialize)
	}

	t.Run("deserialize with invalid base64 SecureString", func(t *testing.T) {
		ss := SecureString("not-valid-base64!@#$%^")
		if _, err := Deserialize[string](kcsValid, ss); err == nil {
			t.Fatalf("expected error for invalid base64 SecureString")
		}
	})

	t.Run("deserialize tampered ciphertext", func(t *testing.T) {
		ciphertext, err := bytesutil.FromBase64(string(ssValid))
		if err != nil {
			t.Fatalf("Setup: FromBase64 for tampering test failed: %v", err)
		}

		if len(ciphertext) == 0 {
			t.Skip("Skipping tamper test for zero-length ciphertext, which is unexpected.")
		}

		tamperedCiphertext := make([]byte, len(ciphertext))
		copy(tamperedCiphertext, ciphertext)

		idxToTamper := len(tamperedCiphertext) / 2
		tamperedCiphertext[idxToTamper] = tamperedCiphertext[idxToTamper] ^ 0x01

		tamperedSS := SecureString(bytesutil.ToBase64(tamperedCiphertext))

		if _, err := Deserialize[string](kcsValid, tamperedSS); err == nil {
			t.Fatalf("expected error for tampered ciphertext")
		}
	})

	t.Run("deserialize with no keys", func(t *testing.T) {
		var noKeys []cryptoutil.Key32
		if _, err := Deserialize[string](noKeys, ssValid); err == nil {
			t.Fatalf("expected error when deserializing with no keys")
		}
	})

	t.Run("deserialize with only nil keys", func(t *testing.T) {
		nilKeys := []cryptoutil.Key32{nil, nil}
		if _, err := Deserialize[string](nilKeys, ssValid); err == nil {
			t.Fatalf("expected error for only nil keys, as no valid key would be found")
		}
	})
}

func TestSecureString_Version(t *testing.T) {
	kcs := mustKeys(t, 1)
	if kcs[0] == nil {
		t.Fatal("Test setup: kcs[0] is nil")
	}

	value := "test message for versioning"
	ss, err := Serialize(kcs, value)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	ciphertext, err := bytesutil.FromBase64(string(ss))
	if err != nil {
		t.Fatalf("FromBase64 failed: %v", err)
	}

	plaintext, err := cryptoutil.DecryptSymmetricXChaCha20Poly1305(ciphertext, kcs[0])
	if err != nil {
		t.Fatalf("Manual DecryptSymmetricXChaCha20Poly1305 failed: %v", err)
	}

	if len(plaintext) == 0 {
		t.Fatal("Manual decryption resulted in empty plaintext")
	}
	originalVersion := plaintext[0]
	plaintext[0] = 99

	modifiedCiphertext, err := cryptoutil.EncryptSymmetricXChaCha20Poly1305(plaintext, kcs[0])
	if err != nil {
		t.Fatalf("Manual EncryptSymmetricXChaCha20Poly1305 failed: %v", err)
	}
	modifiedSS := SecureString(bytesutil.ToBase64(modifiedCiphertext))

	_, err = Deserialize[string](kcs, modifiedSS)
	if err == nil {
		t.Fatalf("expected version error when deserializing with modified version byte")
	}
	plaintext[0] = originalVersion
	validCiphertextAgain, _ := cryptoutil.EncryptSymmetricXChaCha20Poly1305(plaintext, kcs[0])
	validSSAgain := SecureString(bytesutil.ToBase64(validCiphertextAgain))
	if _, err := Deserialize[string](kcs, validSSAgain); err != nil {
		t.Fatalf("Failed to deserialize with original version after modification test: %v", err)
	}

}

func TestSecureString_ComplexTypes(t *testing.T) {
	kcs := mustKeys(t, 1)

	t.Run("time serialization", func(t *testing.T) {
		type TimeData struct {
			Created time.Time
			Updated time.Time
		}

		now := time.Now()
		original := TimeData{
			Created: now,
			Updated: now.Add(24 * time.Hour),
		}

		ss, err := Serialize(kcs, original)
		if err != nil {
			t.Fatalf("Serialize failed for TimeData: %v", err)
		}

		var decoded TimeData
		decoded, err = Deserialize[TimeData](kcs, ss)
		if err != nil {
			t.Fatalf("Deserialize failed for TimeData: %v", err)
		}

		if !decoded.Created.Equal(original.Created) {
			t.Errorf("Created time mismatch: want %v, got %v", original.Created, decoded.Created)
		}
		if !decoded.Updated.Equal(original.Updated) {
			t.Errorf("Updated time mismatch: want %v, got %v", original.Updated, decoded.Updated)
		}
	})

	t.Run("channel not serializable", func(t *testing.T) {
		ch := make(chan int)
		if _, err := Serialize(kcs, ch); err == nil {
			t.Fatalf("expected error when serializing channel")
		}
	})

	t.Run("function not serializable", func(t *testing.T) {
		fn := func() {}
		if _, err := Serialize(kcs, fn); err == nil {
			t.Fatalf("expected error when serializing function")
		}
	})
}

func TestSecureString_Concurrency(t *testing.T) {
	kcs := mustKeys(t, 3)

	const numGoroutines = 50
	const iterationsPerGopher = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errChan := make(chan error, numGoroutines*iterationsPerGopher)

	for i := range numGoroutines {
		go func(gopherID int) {
			defer wg.Done()
			for j := range iterationsPerGopher {
				value := fmt.Sprintf("concurrent-test-gopher-%d-iter-%d", gopherID, j)
				kcsForGoroutine := kcs

				ss, err := Serialize(kcsForGoroutine, value)
				if err != nil {
					errChan <- fmt.Errorf("goroutine %d: Serialize error: %w", gopherID, err)
					return
				}

				got, err := Deserialize[string](kcsForGoroutine, ss)
				if err != nil {
					errChan <- fmt.Errorf("goroutine %d: Deserialize error: %w", gopherID, err)
					return
				}

				if got != value {
					errChan <- fmt.Errorf("goroutine %d: value mismatch: want %q, got %q", gopherID, value, got)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	var errs []string
	for err := range errChan {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		t.Errorf("Concurrency test failed with %d errors:\n%s", len(errs), strings.Join(errs, "\n"))
	}
}

func TestSecureString_CornerCases(t *testing.T) {
	t.Run("many keys for decryption", func(t *testing.T) {
		numKeys := 20
		keyChain := mustKeys(t, numKeys)
		value := "test with many keys decryption"

		ss, err := Serialize(keyChain, value)
		if err != nil {
			t.Fatalf("Serialize failed: %v", err)
		}

		got, err := Deserialize[string](keyChain, ss)
		if err != nil {
			t.Fatalf("Deserialize failed with many keys: %v", err)
		}
		if got != value {
			t.Fatalf("value mismatch with many keys: want %q, got %q", value, got)
		}

		if numKeys > 1 {
			lastKeyIndex := numKeys - 1
			keysWithLastActive := []cryptoutil.Key32{keyChain[lastKeyIndex]}

			ssLast, err := Serialize(keysWithLastActive, value)
			if err != nil {
				t.Fatalf("Serialize with last key failed: %v", err)
			}

			gotLast, err := Deserialize[string](keyChain, ssLast)
			if err != nil {
				t.Fatalf("Deserialize (last key active) failed: %v", err)
			}
			if gotLast != value {
				t.Fatalf("value mismatch (last key active): want %q, got %q", value, gotLast)
			}
		}
	})

	t.Run("payload nearly max size for Serialize", func(t *testing.T) {
		kcs := mustKeys(t, 1)

		gobOverheadEstimate := 100
		xchachaOverhead := chacha20poly1305.Overhead
		versionByteOverhead := 1
		totalOverheadEstimate := gobOverheadEstimate + xchachaOverhead + versionByteOverhead

		if totalOverheadEstimate >= one_mb_in_bytes {
			t.Skip("Overhead estimate is too large for this test relative to one_mb_in_bytes")
		}

		safePayloadSize := one_mb_in_bytes - totalOverheadEstimate - 1
		if safePayloadSize <= 0 {
			t.Skipf("Calculated safePayloadSize %d is too small, check estimates or one_mb_in_bytes", safePayloadSize)
		}

		largeData := make([]byte, safePayloadSize)
		if _, err := rand.Read(largeData); err != nil {
			t.Fatalf("Failed to generate random data for large payload test: %v", err)
		}

		ss, err := Serialize(kcs, largeData)
		if err != nil {
			t.Fatalf("Failed to serialize large but valid payload (size %d): %v", safePayloadSize, err)
		}
		ciphertext, _ := base64.StdEncoding.DecodeString(string(ss))
		t.Logf("Nearly max size test: payload %d bytes, ciphertext %d bytes (limit %d)", safePayloadSize, len(ciphertext), one_mb_in_bytes)
		if len(ciphertext) > one_mb_in_bytes {
			t.Errorf("Ciphertext for nearly max size payload exceeded limit: len %d", len(ciphertext))
		}
	})
}
