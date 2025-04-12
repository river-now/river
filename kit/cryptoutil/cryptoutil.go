// Package cryptoutil provides utility functions for cryptographic operations.
// It is the consumer's responsibility to ensure that inputs are reasonably
// sized so as to avoid memory exhaustion attacks.
package cryptoutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"github.com/river-now/river/kit/bytesutil"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/auth"
)

type Base64 = string

var (
	ErrSecretKeyIsNil     = errors.New("secret key is nil")
	ErrCipherTextTooShort = errors.New("ciphertext too short")
)

/////////////////////////////////////////////////////////////////////
// SYMMETRIC MESSAGE SIGNING
/////////////////////////////////////////////////////////////////////

// SignSymmetric signs a message using a symmetric key. It is a convenience
// wrapper around the nacl/auth package, which uses HMAC-SHA-512-256.
func SignSymmetric(msg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}
	digest := auth.Sum(msg, secretKey)
	signedMsg := make([]byte, auth.Size+len(msg))
	copy(signedMsg, digest[:])
	copy(signedMsg[auth.Size:], msg)
	return signedMsg, nil
}

// VerifyAndReadSymmetric verifies a signed message using a symmetric key and
// returns the original message. It is a convenience wrapper around the
// nacl/auth package, which uses HMAC-SHA-512-256.
func VerifyAndReadSymmetric(signedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, errors.New("secret key is required")
	}
	if len(signedMsg) < auth.Size {
		return nil, errors.New("invalid signature")
	}
	digest := make([]byte, auth.Size)
	copy(digest, signedMsg[:auth.Size])
	msg := signedMsg[auth.Size:]
	if !auth.Verify(digest, msg, secretKey) {
		return nil, errors.New("invalid signature")
	}
	return msg, nil
}

/////////////////////////////////////////////////////////////////////
// ASYMMETRIC MESSAGE SIGNING
/////////////////////////////////////////////////////////////////////

// VerifyAndReadAsymmetric verifies a signed message using an Ed25519 public key and
// returns the original message.
func VerifyAndReadAsymmetric(signedMsg []byte, publicKey *[32]byte) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New("public key is required")
	}
	if len(signedMsg) < ed25519.SignatureSize {
		return nil, errors.New("message shorter than signature size")
	}

	ok := ed25519.Verify(publicKey[:], signedMsg[ed25519.SignatureSize:], signedMsg[:ed25519.SignatureSize])
	if !ok {
		return nil, errors.New("invalid signature")
	}

	return signedMsg[ed25519.SignatureSize:], nil
}

// VerifyAndReadAsymmetricBase64 verifies a signed message using a base64
// encoded Ed25519 public key and returns the original message.
func VerifyAndReadAsymmetricBase64(signedMsg Base64, publicKey Base64) ([]byte, error) {
	signedMsgBytes, err := bytesutil.FromBase64(signedMsg)
	if err != nil {
		return nil, err
	}

	publicKeyBytes, err := bytesutil.FromBase64(publicKey)
	if err != nil {
		return nil, err
	}
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key size")
	}

	var publicKey32 [32]byte
	copy(publicKey32[:], publicKeyBytes)

	return VerifyAndReadAsymmetric(signedMsgBytes, &publicKey32)
}

/////////////////////////////////////////////////////////////////////
// FAST HASHING
/////////////////////////////////////////////////////////////////////

// Sha256Hash returns the SHA-256 hash of a message as a byte slice.
func Sha256Hash(msg []byte) []byte {
	hash := sha256.Sum256(msg)
	return hash[:]
}

/////////////////////////////////////////////////////////////////////
// SYMMETRIC ENCRYPTION
/////////////////////////////////////////////////////////////////////

// EncryptSymmetricXChaCha20Poly1305 encrypts a message using XChaCha20-Poly1305.
func EncryptSymmetricXChaCha20Poly1305(msg []byte, secretKey *[32]byte) ([]byte, error) {
	return EncryptSymmetricGeneric(ToAEADFuncXChaCha20Poly1305, msg, secretKey)
}

// DecryptSymmetricXChaCha20Poly1305 decrypts a message using XChaCha20-Poly1305.
func DecryptSymmetricXChaCha20Poly1305(encryptedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	return DecryptSymmetricGeneric(ToAEADFuncXChaCha20Poly1305, encryptedMsg, secretKey)
}

// EncryptSymmetricAESGCM encrypts a message using AES-256-GCM.
func EncryptSymmetricAESGCM(msg []byte, secretKey *[32]byte) ([]byte, error) {
	return EncryptSymmetricGeneric(ToAEADFuncAESGCM, msg, secretKey)
}

// DecryptSymmetricAESGCM decrypts a message using AES-256-GCM.
func DecryptSymmetricAESGCM(encryptedMsg []byte, secretKey *[32]byte) ([]byte, error) {
	return DecryptSymmetricGeneric(ToAEADFuncAESGCM, encryptedMsg, secretKey)
}

// ToAEADFuncXChaCha20Poly1305 returns an AEAD function for XChaCha20-Poly1305.
var ToAEADFuncXChaCha20Poly1305 ToAEADFunc = func(secretKey *[32]byte) (cipher.AEAD, error) {
	return chacha20poly1305.NewX(secretKey[:])
}

// ToAEADFuncAESGCM returns an AEAD function for AES-256-GCM.
var ToAEADFuncAESGCM ToAEADFunc = func(secretKey *[32]byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(secretKey[:])
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(block)
}

type ToAEADFunc func(secretKey *[32]byte) (cipher.AEAD, error)

// EncryptSymmetricGeneric encrypts a message using a generic AEAD function.
func EncryptSymmetricGeneric(toAEADFunc ToAEADFunc, msg []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, ErrSecretKeyIsNil
	}

	aead, err := toAEADFunc(secretKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return aead.Seal(nonce, nonce, msg, nil), nil
}

// DecryptSymmetricGeneric decrypts a message using a generic AEAD function.
func DecryptSymmetricGeneric(toAEADFunc ToAEADFunc, ciphertext []byte, secretKey *[32]byte) ([]byte, error) {
	if secretKey == nil {
		return nil, ErrSecretKeyIsNil
	}

	aead, err := toAEADFunc(secretKey)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()

	if len(ciphertext) < nonceSize+aead.Overhead() {
		return nil, ErrCipherTextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	return aead.Open(nil, nonce, ciphertext, nil)
}
