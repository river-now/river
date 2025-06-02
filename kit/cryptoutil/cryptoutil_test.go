package cryptoutil

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"golang.org/x/crypto/nacl/auth"
)

const (
	aesNonceSize               = 12 // Size of AES-GCM nonce
	xChaCha20Poly1305NonceSize = 24 // Size of XChaCha20-Poly1305 nonce
)

func new32() *[32]byte {
	return &[32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func TestSignSymmetric(t *testing.T) {
	secretKey := new32()
	message := []byte("test message")

	signedMsg, err := SignSymmetric(message, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(signedMsg) != auth.Size+len(message) {
		t.Fatalf("expected signed message length %d, got %d", auth.Size+len(message), len(signedMsg))
	}

	// Test that the signed message contains the original message
	if !bytes.Equal(signedMsg[auth.Size:], message) {
		t.Fatalf("expected signed message to contain original message")
	}
}

func TestVerifyAndReadSymmetric(t *testing.T) {
	secretKey := new32()
	message := []byte("test message")

	signedMsg, _ := SignSymmetric(message, secretKey)

	// Successful verification
	retrievedMsg, err := VerifyAndReadSymmetric(signedMsg, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(retrievedMsg, message) {
		t.Fatalf("expected retrieved message to equal original message")
	}

	// Invalid signature (corrupt the signed message)
	signedMsg[0] ^= 0xFF // flip a bit in the signature
	_, err = VerifyAndReadSymmetric(signedMsg, secretKey)
	if err == nil {
		t.Fatalf("expected error due to invalid signature, got nil")
	}

	// Truncated message
	truncatedMsg := signedMsg[:auth.Size-1]
	_, err = VerifyAndReadSymmetric(truncatedMsg, secretKey)
	if err == nil {
		t.Fatalf("expected error due to truncated message, got nil")
	}
}

func TestVerifyAndReadAsymmetric(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	message := []byte("test message")
	signature := ed25519.Sign(privateKey, message)
	signedMsg := append(signature, message...)

	// Convert public key to [32]byte format
	var publicKey32 [32]byte
	copy(publicKey32[:], publicKey)

	// Successful verification
	retrievedMsg, err := VerifyAndReadAsymmetric(signedMsg, &publicKey32)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(retrievedMsg, message) {
		t.Fatalf("expected retrieved message to equal original message")
	}

	// Invalid signature (corrupt the signed message)
	signedMsg[0] ^= 0xFF // flip a bit in the signature
	_, err = VerifyAndReadAsymmetric(signedMsg, &publicKey32)
	if err == nil {
		t.Fatalf("expected error due to invalid signature, got nil")
	}

	// Truncated message
	truncatedMsg := signedMsg[:len(signedMsg)-1]
	_, err = VerifyAndReadAsymmetric(truncatedMsg, &publicKey32)
	if err == nil {
		t.Fatalf("expected error due to truncated message, got nil")
	}
}

func TestVerifyAndReadAsymmetricBase64(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	message := []byte("test message")
	signature := ed25519.Sign(privateKey, message)
	signedMsg := append(signature, message...)

	signedMsgBase64 := base64.StdEncoding.EncodeToString(signedMsg)
	publicKeyBase64 := base64.StdEncoding.EncodeToString(publicKey)

	// Successful verification
	retrievedMsg, err := VerifyAndReadAsymmetricBase64(signedMsgBase64, publicKeyBase64)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !bytes.Equal(retrievedMsg, message) {
		t.Fatalf("expected retrieved message to equal original message")
	}

	// Invalid base64 signature
	_, err = VerifyAndReadAsymmetricBase64("invalid_base64", publicKeyBase64)
	if err == nil {
		t.Fatalf("expected error due to invalid base64 signature, got nil")
	}

	// Invalid base64 public key
	_, err = VerifyAndReadAsymmetricBase64(signedMsgBase64, "invalid_base64")
	if err == nil {
		t.Fatalf("expected error due to invalid base64 public key, got nil")
	}

	// Invalid signature (corrupt the signed message)
	tampered := make([]byte, len(signedMsg))
	copy(tampered, signedMsg)
	tampered[0] ^= 0xFF
	tamperedBase64 := base64.StdEncoding.EncodeToString(tampered)
	_, err = VerifyAndReadAsymmetricBase64(tamperedBase64, publicKeyBase64)
	if err == nil {
		t.Fatalf("expected error due to invalid signature, got nil")
	}
}

func TestEdgeCases(t *testing.T) {
	secretKey := new32()
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	var publicKey32 [32]byte
	copy(publicKey32[:], publicKey)

	// Empty message for symmetric signing
	signedMsg, err := SignSymmetric([]byte{}, secretKey)
	if err != nil {
		t.Fatalf("expected no error for empty message, got %v", err)
	}
	if len(signedMsg) != auth.Size {
		t.Fatalf("expected signed message length %d, got %d", auth.Size, len(signedMsg))
	}

	// Empty signed message for symmetric verification
	_, err = VerifyAndReadSymmetric([]byte{}, secretKey)
	if err == nil {
		t.Fatalf("expected error due to empty signed message, got nil")
	}

	// Empty signed message for asymmetric verification
	_, err = VerifyAndReadAsymmetric([]byte{}, &publicKey32)
	if err == nil {
		t.Fatalf("expected error due to empty signed message, got nil")
	}

	// Nil secret key for symmetric signing
	_, err = SignSymmetric([]byte("test"), nil)
	if err == nil {
		t.Fatalf("expected error due to nil secret key, got nil")
	}

	// Nil secret key for symmetric verification
	_, err = VerifyAndReadSymmetric([]byte("test"), nil)
	if err == nil {
		t.Fatalf("expected error due to nil secret key, got nil")
	}

	// Nil public key for asymmetric verification
	_, err = VerifyAndReadAsymmetric([]byte("test"), nil)
	if err == nil {
		t.Fatalf("expected error due to nil public key, got nil")
	}
}

func TestEncryptSymmetricAESGCM(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for encryption")

	// Test successful encryption
	encrypted, err := EncryptSymmetricAESGCM(message, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(encrypted) <= aesNonceSize {
		t.Fatalf("expected encrypted message to be longer than nonce, got length %d", len(encrypted))
	}

	// Test that encrypted message is different from original
	if bytes.Equal(encrypted[aesNonceSize:], message) {
		t.Fatalf("encrypted message should not be equal to original message")
	}

	// Test encryption with nil secret key
	_, err = EncryptSymmetricAESGCM(message, nil)
	if err == nil {
		t.Fatalf("expected error with nil secret key, got nil")
	}

	// Test encryption of empty message
	emptyEncrypted, err := EncryptSymmetricAESGCM([]byte{}, secretKey)
	if err != nil {
		t.Fatalf("expected no error for empty message, got %v", err)
	}
	if len(emptyEncrypted) <= aesNonceSize {
		t.Fatalf("expected encrypted empty message to be longer than nonce, got length %d", len(emptyEncrypted))
	}
}

func TestDecryptSymmetricAESGCM(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for decryption")

	// Test successful encryption and decryption
	encrypted, _ := EncryptSymmetricAESGCM(message, secretKey)
	decrypted, err := DecryptSymmetricAESGCM(encrypted, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(decrypted, message) {
		t.Fatalf("decrypted message does not match original")
	}

	// Test decryption with wrong key
	wrongKey := new32()
	wrongKey[0] ^= 0xFF // Flip a bit to make it different
	_, err = DecryptSymmetricAESGCM(encrypted, wrongKey)
	if err == nil {
		t.Fatalf("expected error with wrong key, got nil")
	}

	// Test decryption of tampered message
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0xFF // Flip the last bit
	_, err = DecryptSymmetricAESGCM(tampered, secretKey)
	if err == nil {
		t.Fatalf("expected error with tampered message, got nil")
	}

	// Test decryption with nil secret key
	_, err = DecryptSymmetricAESGCM(encrypted, nil)
	if err == nil {
		t.Fatalf("expected error with nil secret key, got nil")
	}

	// Test decryption of message that's too short
	_, err = DecryptSymmetricAESGCM(encrypted[:aesNonceSize-1], secretKey)
	if err == nil {
		t.Fatalf("expected error with short message, got nil")
	}

	// Test decryption of empty message
	_, err = DecryptSymmetricAESGCM([]byte{}, secretKey)
	if err == nil {
		t.Fatalf("expected error with empty message, got nil")
	}
}

func TestEncryptSymmetricXChaCha20(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for encryption")

	// Test successful encryption
	encrypted, err := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(encrypted) <= xChaCha20Poly1305NonceSize {
		t.Fatalf("expected encrypted message to be longer than nonce, got length %d", len(encrypted))
	}

	// Test that encrypted message is different from original
	if bytes.Equal(encrypted[xChaCha20Poly1305NonceSize:], message) {
		t.Fatalf("encrypted message should not be equal to original message")
	}

	// Test encryption with nil secret key
	_, err = EncryptSymmetricXChaCha20Poly1305(message, nil)
	if err == nil {
		t.Fatalf("expected error with nil secret key, got nil")
	}

	// Test encryption of empty message
	emptyEncrypted, err := EncryptSymmetricXChaCha20Poly1305([]byte{}, secretKey)
	if err != nil {
		t.Fatalf("expected no error for empty message, got %v", err)
	}
	if len(emptyEncrypted) <= xChaCha20Poly1305NonceSize {
		t.Fatalf("expected encrypted empty message to be longer than nonce, got length %d", len(emptyEncrypted))
	}
}

func TestDecryptSymmetricXChaCha20(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for decryption")

	// Test successful encryption and decryption
	encrypted, _ := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	decrypted, err := DecryptSymmetricXChaCha20Poly1305(encrypted, secretKey)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !bytes.Equal(decrypted, message) {
		t.Fatalf("decrypted message does not match original")
	}

	// Test decryption with wrong key
	wrongKey := new32()
	wrongKey[0] ^= 0xFF // Flip a bit to make it different
	_, err = DecryptSymmetricXChaCha20Poly1305(encrypted, wrongKey)
	if err == nil {
		t.Fatalf("expected error with wrong key, got nil")
	}

	// Test decryption of tampered message
	tampered := make([]byte, len(encrypted))
	copy(tampered, encrypted)
	tampered[len(tampered)-1] ^= 0xFF // Flip the last bit
	_, err = DecryptSymmetricXChaCha20Poly1305(tampered, secretKey)
	if err == nil {
		t.Fatalf("expected error with tampered message, got nil")
	}

	// Test decryption with nil secret key
	_, err = DecryptSymmetricXChaCha20Poly1305(encrypted, nil)
	if err == nil {
		t.Fatalf("expected error with nil secret key, got nil")
	}

	// Test decryption of message that's too short
	_, err = DecryptSymmetricXChaCha20Poly1305(encrypted[:xChaCha20Poly1305NonceSize-1], secretKey)
	if err == nil {
		t.Fatalf("expected error with short message, got nil")
	}

	// Test decryption of empty message
	_, err = DecryptSymmetricXChaCha20Poly1305([]byte{}, secretKey)
	if err == nil {
		t.Fatalf("expected error with empty message, got nil")
	}
}

func TestCrossEncryptionCompatibility(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for cross-compatibility")

	// Test that AES-GCM encrypted messages can't be decrypted by XChaCha20
	encryptedAES, _ := EncryptSymmetricAESGCM(message, secretKey)
	_, err := DecryptSymmetricXChaCha20Poly1305(encryptedAES, secretKey)
	if err == nil {
		t.Fatalf("expected error decrypting AES-GCM message with XChaCha20, got nil")
	}

	// Test that XChaCha20 encrypted messages can't be decrypted by AES-GCM
	encryptedXChaCha, _ := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	_, err = DecryptSymmetricAESGCM(encryptedXChaCha, secretKey)
	if err == nil {
		t.Fatalf("expected error decrypting XChaCha20 message with AES-GCM, got nil")
	}
}

func TestReplayProtection(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for replay protection")

	// Test AES-GCM replay protection (nonce reuse)
	encrypted1, _ := EncryptSymmetricAESGCM(message, secretKey)
	encrypted2, _ := EncryptSymmetricAESGCM(message, secretKey)
	if bytes.Equal(encrypted1, encrypted2) {
		t.Fatalf("AES-GCM: two encryptions of same message should produce different ciphertexts")
	}

	// Test XChaCha20 replay protection (nonce reuse)
	encrypted3, _ := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	encrypted4, _ := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	if bytes.Equal(encrypted3, encrypted4) {
		t.Fatalf("XChaCha20: two encryptions of same message should produce different ciphertexts")
	}
}

func TestConcurrentEncryption(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for concurrent encryption")
	iterations := 100

	// Test concurrent AES-GCM encryption
	done := make(chan bool)
	for i := 0; i < iterations; i++ {
		go func() {
			encrypted, err := EncryptSymmetricAESGCM(message, secretKey)
			if err != nil {
				t.Errorf("concurrent AES-GCM encryption failed: %v", err)
			}
			decrypted, err := DecryptSymmetricAESGCM(encrypted, secretKey)
			if err != nil {
				t.Errorf("concurrent AES-GCM decryption failed: %v", err)
			}
			if !bytes.Equal(decrypted, message) {
				t.Errorf("concurrent AES-GCM decrypted message does not match original")
			}
			done <- true
		}()
	}

	// Wait for all AES-GCM goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}

	// Test concurrent XChaCha20 encryption
	for i := 0; i < iterations; i++ {
		go func() {
			encrypted, err := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
			if err != nil {
				t.Errorf("concurrent XChaCha20 encryption failed: %v", err)
			}
			decrypted, err := DecryptSymmetricXChaCha20Poly1305(encrypted, secretKey)
			if err != nil {
				t.Errorf("concurrent XChaCha20 decryption failed: %v", err)
			}
			if !bytes.Equal(decrypted, message) {
				t.Errorf("concurrent XChaCha20 decrypted message does not match original")
			}
			done <- true
		}()
	}

	// Wait for all XChaCha20 goroutines
	for i := 0; i < iterations; i++ {
		<-done
	}
}

func TestNonceUniqueness(t *testing.T) {
	secretKey := new32()
	message := []byte("test message for nonce uniqueness")
	nonceCount := 1000
	aesNonces := make(map[string]bool)
	chachaNonces := make(map[string]bool)

	// Test XChaCha20Poly1305 nonce uniqueness
	for i := 0; i < nonceCount; i++ {
		encrypted, err := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
		if err != nil {
			t.Fatalf("XChaCha20Poly1305 encryption failed: %v", err)
		}
		nonce := string(encrypted[:xChaCha20Poly1305NonceSize])
		if chachaNonces[nonce] {
			t.Fatalf("XChaCha20Poly1305: duplicate nonce detected after %d iterations", i)
		}
		chachaNonces[nonce] = true
	}

	// Test AES-GCM nonce uniqueness
	for i := 0; i < nonceCount; i++ {
		encrypted, err := EncryptSymmetricAESGCM(message, secretKey)
		if err != nil {
			t.Fatalf("AES-GCM encryption failed: %v", err)
		}
		nonce := string(encrypted[:aesNonceSize])
		if aesNonces[nonce] {
			t.Fatalf("AES-GCM: duplicate nonce detected after %d iterations", i)
		}
		aesNonces[nonce] = true
	}
}

func TestLargeMessageEncryption(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large message test in short mode")
	}

	secretKey := new32()
	sizes := []int{
		64 * 1024,        // 64 KB
		1 * 1024 * 1024,  // 1 MB
		16 * 1024 * 1024, // 16 MB - tests chunking/buffering behavior
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			largeMessage := make([]byte, size)
			if _, err := rand.Read(largeMessage); err != nil {
				t.Fatalf("failed to generate random message: %v", err)
			}

			// Test XChaCha20Poly1305
			encrypted1, err := EncryptSymmetricXChaCha20Poly1305(largeMessage, secretKey)
			if err != nil {
				t.Fatalf("XChaCha20Poly1305 encryption failed: %v", err)
			}
			decrypted1, err := DecryptSymmetricXChaCha20Poly1305(encrypted1, secretKey)
			if err != nil {
				t.Fatalf("XChaCha20Poly1305 decryption failed: %v", err)
			}
			if !bytes.Equal(decrypted1, largeMessage) {
				t.Fatal("XChaCha20Poly1305: decrypted message does not match original")
			}

			// Test AES-GCM with same data
			encrypted2, err := EncryptSymmetricAESGCM(largeMessage, secretKey)
			if err != nil {
				t.Fatalf("AES-GCM encryption failed: %v", err)
			}
			decrypted2, err := DecryptSymmetricAESGCM(encrypted2, secretKey)
			if err != nil {
				t.Fatalf("AES-GCM decryption failed: %v", err)
			}
			if !bytes.Equal(decrypted2, largeMessage) {
				t.Fatal("AES-GCM: decrypted message does not match original")
			}
		})
	}
}

func TestTinyMessages(t *testing.T) {
	secretKey := new32()
	messages := [][]byte{
		{},           // empty
		{0},          // single byte
		{1, 2},       // two bytes
		{1, 2, 3, 4}, // few bytes
	}

	for _, msg := range messages {
		// Test XChaCha20Poly1305
		encrypted1, err := EncryptSymmetricXChaCha20Poly1305(msg, secretKey)
		if err != nil {
			t.Errorf("XChaCha20Poly1305 failed to encrypt %d byte message: %v", len(msg), err)
		}
		decrypted1, err := DecryptSymmetricXChaCha20Poly1305(encrypted1, secretKey)
		if err != nil {
			t.Errorf("XChaCha20Poly1305 failed to decrypt %d byte message: %v", len(msg), err)
		}
		if !bytes.Equal(decrypted1, msg) {
			t.Errorf("XChaCha20Poly1305 failed to round-trip %d byte message", len(msg))
		}

		// Test AES-GCM
		encrypted2, err := EncryptSymmetricAESGCM(msg, secretKey)
		if err != nil {
			t.Errorf("AES-GCM failed to encrypt %d byte message: %v", len(msg), err)
		}
		decrypted2, err := DecryptSymmetricAESGCM(encrypted2, secretKey)
		if err != nil {
			t.Errorf("AES-GCM failed to decrypt %d byte message: %v", len(msg), err)
		}
		if !bytes.Equal(decrypted2, msg) {
			t.Errorf("AES-GCM failed to round-trip %d byte message", len(msg))
		}
	}
}

func TestAuthenticationTag(t *testing.T) {
	secretKey := new32()
	message := []byte("test message")

	// Test XChaCha20Poly1305 authentication
	encrypted1, err := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	if err != nil {
		t.Fatal(err)
	}
	// Modify each byte at the end (where auth tag likely is)
	for i := 1; i <= 16; i++ {
		corrupt := make([]byte, len(encrypted1))
		copy(corrupt, encrypted1)
		corrupt[len(corrupt)-i] ^= 0x1
		_, err := DecryptSymmetricXChaCha20Poly1305(corrupt, secretKey)
		if err == nil {
			t.Errorf("XChaCha20Poly1305 did not detect corruption at byte -%d", i)
		}
	}

	// Test AES-GCM authentication
	encrypted2, err := EncryptSymmetricAESGCM(message, secretKey)
	if err != nil {
		t.Fatal(err)
	}
	// Modify each byte at the end (where auth tag likely is)
	for i := 1; i <= 16; i++ {
		corrupt := make([]byte, len(encrypted2))
		copy(corrupt, encrypted2)
		corrupt[len(corrupt)-i] ^= 0x1
		_, err := DecryptSymmetricAESGCM(corrupt, secretKey)
		if err == nil {
			t.Errorf("AES-GCM did not detect corruption at byte -%d", i)
		}
	}
}

func TestInvalidKeySize(t *testing.T) {
	// Test truncated public key
	shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16)) // too short
	longKey := base64.StdEncoding.EncodeToString(make([]byte, 64))  // too long
	message := []byte("test")

	_, err := VerifyAndReadAsymmetricBase64(base64.StdEncoding.EncodeToString(message), shortKey)
	if err == nil {
		t.Error("expected error with short public key")
	}

	_, err = VerifyAndReadAsymmetricBase64(base64.StdEncoding.EncodeToString(message), longKey)
	if err == nil {
		t.Error("expected error with long public key")
	}

	_, err = VerifyAndReadAsymmetricBase64(base64.StdEncoding.EncodeToString(message), "")
	if err == nil {
		t.Error("expected error with empty public key")
	}
}

func TestEmptyInputs(t *testing.T) {
	secretKey := new32()
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	var publicKey32 [32]byte
	copy(publicKey32[:], publicKey)

	// Test nil/empty byte slices for each function
	// SignSymmetric still needs to work with empty message
	signed, err := SignSymmetric(nil, secretKey)
	if err != nil {
		t.Errorf("SignSymmetric failed with nil message: %v", err)
	}
	if len(signed) != auth.Size {
		t.Errorf("SignSymmetric with nil message: expected length %d, got %d", auth.Size, len(signed))
	}

	// Test signature operations with empty string base64
	_, err = VerifyAndReadAsymmetricBase64("", base64.StdEncoding.EncodeToString(publicKey))
	if err == nil {
		t.Error("expected error with empty signed message")
	}

	// Test nil/empty inputs for encryption
	for _, f := range []struct {
		name    string
		encrypt func([]byte, Key32) ([]byte, error)
	}{
		{"XChaCha20Poly1305", EncryptSymmetricXChaCha20Poly1305},
		{"AESGCM", EncryptSymmetricAESGCM},
	} {
		t.Run(f.name, func(t *testing.T) {
			// nil message should work (encrypting empty message)
			encrypted, err := f.encrypt(nil, secretKey)
			if err != nil {
				t.Errorf("encryption failed with nil message: %v", err)
			}
			if len(encrypted) < 16 { // at least nonce + tag
				t.Error("encrypted output too short")
			}
		})
	}
}

func TestExtremeMessageSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extreme size tests in short mode")
	}

	secretKey := new32()
	// Test various message sizes including edge cases
	sizes := []int{
		0,             // empty
		1,             // minimum
		15,            // below typical block size
		16,            // typical block size
		17,            // above typical block size
		1024*1024 - 1, // just below 1MB
		1024 * 1024,   // exactly 1MB
		1024*1024 + 1, // just above 1MB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size-%d", size), func(t *testing.T) {
			msg := make([]byte, size)
			rand.Read(msg)

			// Test both encryption methods
			encrypted1, err := EncryptSymmetricXChaCha20Poly1305(msg, secretKey)
			if err != nil {
				t.Errorf("XChaCha20Poly1305 failed with size %d: %v", size, err)
			}
			decrypted1, err := DecryptSymmetricXChaCha20Poly1305(encrypted1, secretKey)
			if err != nil || !bytes.Equal(decrypted1, msg) {
				t.Errorf("XChaCha20Poly1305 round-trip failed with size %d", size)
			}

			encrypted2, err := EncryptSymmetricAESGCM(msg, secretKey)
			if err != nil {
				t.Errorf("AESGCM failed with size %d: %v", size, err)
			}
			decrypted2, err := DecryptSymmetricAESGCM(encrypted2, secretKey)
			if err != nil || !bytes.Equal(decrypted2, msg) {
				t.Errorf("AESGCM round-trip failed with size %d", size)
			}
		})
	}
}

func TestInputValidation(t *testing.T) {
	secretKey := new32()
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	var publicKey32 [32]byte
	copy(publicKey32[:], publicKey)
	message := []byte("test message")

	// Prepare some valid encrypted/signed messages for testing
	validEncryptedAESGCM, _ := EncryptSymmetricAESGCM(message, secretKey)
	validEncryptedXChaCha, _ := EncryptSymmetricXChaCha20Poly1305(message, secretKey)
	validSignedSymmetric, _ := SignSymmetric(message, secretKey)
	validSignedAsymmetric := append(ed25519.Sign(privateKey, message), message...)

	tests := []struct {
		name    string
		f       func() error
		wantErr bool
		errMsg  string
	}{
		// SignSymmetric validation
		{
			name: "SignSymmetric nil key",
			f: func() error {
				_, err := SignSymmetric(message, nil)
				return err
			},
			wantErr: true,
			errMsg:  "secret key is required",
		},
		{
			name: "SignSymmetric nil message",
			f: func() error {
				_, err := SignSymmetric(nil, secretKey)
				return err
			},
			wantErr: false,
		},

		// VerifyAndReadSymmetric validation
		{
			name: "VerifyAndReadSymmetric nil key",
			f: func() error {
				_, err := VerifyAndReadSymmetric(validSignedSymmetric, nil)
				return err
			},
			wantErr: true,
			errMsg:  "secret key is required",
		},
		{
			name: "VerifyAndReadSymmetric short message",
			f: func() error {
				_, err := VerifyAndReadSymmetric(make([]byte, auth.Size-1), secretKey)
				return err
			},
			wantErr: true,
			errMsg:  "invalid signature",
		},

		// VerifyAndReadAsymmetric validation
		{
			name: "VerifyAndReadAsymmetric nil key",
			f: func() error {
				_, err := VerifyAndReadAsymmetric(validSignedAsymmetric, nil)
				return err
			},
			wantErr: true,
			errMsg:  "public key is required",
		},
		{
			name: "VerifyAndReadAsymmetric short message",
			f: func() error {
				_, err := VerifyAndReadAsymmetric(make([]byte, ed25519.SignatureSize-1), &publicKey32)
				return err
			},
			wantErr: true,
			errMsg:  "message shorter than signature size",
		},

		// VerifyAndReadAsymmetricBase64 validation
		{
			name: "VerifyAndReadAsymmetricBase64 invalid base64 message",
			f: func() error {
				_, err := VerifyAndReadAsymmetricBase64("invalid-base64", base64.StdEncoding.EncodeToString(publicKey))
				return err
			},
			wantErr: true,
		},
		{
			name: "VerifyAndReadAsymmetricBase64 invalid base64 key",
			f: func() error {
				_, err := VerifyAndReadAsymmetricBase64(base64.StdEncoding.EncodeToString(validSignedAsymmetric), "invalid-base64")
				return err
			},
			wantErr: true,
		},
		{
			name: "VerifyAndReadAsymmetricBase64 wrong key size",
			f: func() error {
				wrongSizeKey := make([]byte, 31) // Not 32 bytes
				_, err := VerifyAndReadAsymmetricBase64(
					base64.StdEncoding.EncodeToString(validSignedAsymmetric),
					base64.StdEncoding.EncodeToString(wrongSizeKey),
				)
				return err
			},
			wantErr: true,
			errMsg:  "invalid public key size",
		},

		// EncryptSymmetricXChaCha20Poly1305 validation
		{
			name: "EncryptSymmetricXChaCha20Poly1305 nil key",
			f: func() error {
				_, err := EncryptSymmetricXChaCha20Poly1305(message, nil)
				return err
			},
			wantErr: true,
			errMsg:  ErrSecretKeyIsNil.Error(),
		},
		{
			name: "EncryptSymmetricXChaCha20Poly1305 nil message",
			f: func() error {
				_, err := EncryptSymmetricXChaCha20Poly1305(nil, secretKey)
				return err
			},
			wantErr: false,
		},

		// DecryptSymmetricXChaCha20Poly1305 validation
		{
			name: "DecryptSymmetricXChaCha20Poly1305 nil key",
			f: func() error {
				_, err := DecryptSymmetricXChaCha20Poly1305(validEncryptedXChaCha, nil)
				return err
			},
			wantErr: true,
			errMsg:  ErrSecretKeyIsNil.Error(),
		},
		{
			name: "DecryptSymmetricXChaCha20Poly1305 short message (< nonce)",
			f: func() error {
				_, err := DecryptSymmetricXChaCha20Poly1305(make([]byte, xChaCha20Poly1305NonceSize-1), secretKey)
				return err
			},
			wantErr: true,
			errMsg:  ErrCipherTextTooShort.Error(),
		},
		{
			name: "DecryptSymmetricXChaCha20Poly1305 short message (no tag)",
			f: func() error {
				_, err := DecryptSymmetricXChaCha20Poly1305(make([]byte, xChaCha20Poly1305NonceSize+15), secretKey)
				return err
			},
			wantErr: true,
			errMsg:  ErrCipherTextTooShort.Error(),
		},

		// EncryptSymmetricAESGCM validation
		{
			name: "EncryptSymmetricAESGCM nil key",
			f: func() error {
				_, err := EncryptSymmetricAESGCM(message, nil)
				return err
			},
			wantErr: true,
			errMsg:  ErrSecretKeyIsNil.Error(),
		},
		{
			name: "EncryptSymmetricAESGCM nil message",
			f: func() error {
				_, err := EncryptSymmetricAESGCM(nil, secretKey)
				return err
			},
			wantErr: false,
		},

		// DecryptSymmetricAESGCM validation
		{
			name: "DecryptSymmetricAESGCM nil key",
			f: func() error {
				_, err := DecryptSymmetricAESGCM(validEncryptedAESGCM, nil)
				return err
			},
			wantErr: true,
			errMsg:  ErrSecretKeyIsNil.Error(),
		},
		{
			name: "DecryptSymmetricAESGCM short message (< nonce)",
			f: func() error {
				_, err := DecryptSymmetricAESGCM(make([]byte, aesNonceSize-1), secretKey)
				return err
			},
			wantErr: true,
			errMsg:  ErrCipherTextTooShort.Error(),
		},
		{
			name: "DecryptSymmetricAESGCM short message (no tag)",
			f: func() error {
				_, err := DecryptSymmetricAESGCM(make([]byte, aesNonceSize+15), secretKey)
				return err
			},
			wantErr: true,
			errMsg:  ErrCipherTextTooShort.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.f()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
