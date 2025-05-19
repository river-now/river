// Package bytesutil provides utility functions for byte slice operations.
package bytesutil

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"reflect"
)

// Random returns a slice of cryptographically random bytes of length byteLen.
func Random(byteLen int) ([]byte, error) {
	r := make([]byte, byteLen)
	if _, err := rand.Read(r); err != nil {
		return nil, err
	}
	return r, nil
}

// FromBase64 decodes a base64-encoded string into a byte slice.
func FromBase64(base64Str string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(base64Str)
}

// FromBase64Multi decodes multiple base64-encoded strings into byte slices.
func FromBase64Multi(base64Strs ...string) ([][]byte, error) {
	var bytes = make([][]byte, 0, len(base64Strs))
	for _, str := range base64Strs {
		b, err := FromBase64(str)
		if err != nil {
			return nil, err
		}
		bytes = append(bytes, b)
	}
	return bytes, nil
}

// ToBase64 encodes a byte slice into a base64-encoded string.
func ToBase64(bytes []byte) string {
	return base64.StdEncoding.EncodeToString(bytes)
}

// ToBase64Multi encodes multiple byte slices into base64-encoded strings.
func ToBase64Multi(bytes ...[]byte) []string {
	var strs = make([]string, 0, len(bytes))
	for _, b := range bytes {
		strs = append(strs, ToBase64(b))
	}
	return strs
}

// ToGob encodes an arbitrary value into a gob-encoded byte slice.
func ToGob(src any) ([]byte, error) {
	rv := reflect.ValueOf(src)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil, fmt.Errorf("bytesutil.ToGob: cannot encode nil pointer value")
	}
	var a bytes.Buffer
	enc := gob.NewEncoder(&a)
	err := enc.Encode(src)
	if err != nil {
		return nil, fmt.Errorf("bytesutil.ToGob: failed to encode src to bytes: %w", err)
	}
	return a.Bytes(), nil
}

// FromGobInto decodes a gob-encoded byte slice into a destination.
// The destination must be a pointer to the destination type.
func FromGobInto(gobBytes []byte, destPtr any) error {
	if gobBytes == nil {
		return fmt.Errorf("bytesutil.FromGobInto: cannot decode nil bytes")
	}
	if destPtr == nil {
		return fmt.Errorf("bytesutil.FromGobInto: cannot decode into nil destination")
	}
	dec := gob.NewDecoder(bytes.NewReader(gobBytes))
	err := dec.Decode(destPtr)
	if err != nil {
		return fmt.Errorf("bytesutil.FromGobInto: failed to decode bytes into dest: %w", err)
	}
	return nil
}
