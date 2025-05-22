package id

import (
	"strings"
	"testing"
)

func TestIDNew(t *testing.T) {
	for i := 0; i <= 255; i++ {
		id, err := New(uint8(i))

		// ensure no error
		if err != nil {
			t.Errorf("New() returned error: %v", err)
			continue
		}

		// ensure correct length
		if len(id) != i {
			t.Errorf("New() returned ID of length %d, expected %d", len(id), i)
			continue
		}

		// ensure no invalid characters
		if strings.ContainsAny(id, "-_+/=") {
			t.Errorf("New() returned ID with invalid characters: %s", id)
		}
	}
}

func TestIDNewEdgeCases(t *testing.T) {
	// Test with idLen = 0
	id, err := New(0)
	if err != nil {
		t.Errorf("New(0) returned error: %v", err)
	}
	if id != "" {
		t.Errorf("New(0) returned non-empty ID: %s", id)
	}

	// Test with idLen = 255
	id, err = New(255)
	if err != nil {
		t.Errorf("New(255) returned error: %v", err)
	}
	if len(id) != 255 {
		t.Errorf("New(255) returned ID of length %d, expected 255", len(id))
	}
}

func TestNewMulti(t *testing.T) {
	// Test with count = 0
	ids, err := NewMulti(10, 0)
	if err != nil {
		t.Errorf("NewMulti(10, 0) returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("NewMulti(10, 0) returned non-empty slice: %v", ids)
	}

	// Test with valid idLen and count
	ids, err = NewMulti(10, 5)
	if err != nil {
		t.Errorf("NewMulti(10, 5) returned error: %v", err)
	}
	if len(ids) != 5 {
		t.Errorf("NewMulti(10, 5) returned slice of length %d, expected 5", len(ids))
	}
	for _, id := range ids {
		if len(id) != 10 {
			t.Errorf("NewMulti() returned ID of length %d, expected 10", len(id))
		}
	}
}

func TestIDRandomness(t *testing.T) {
	// Test for randomness
	id1, _ := New(10)
	id2, _ := New(10)
	if id1 == id2 {
		t.Errorf("New() returned identical IDs: %s and %s", id1, id2)
	}
}

func TestNewCustomCharset(t *testing.T) {
	// Test with numeric-only charset
	numericCharset := "0123456789"
	id, err := New(10, numericCharset)
	if err != nil {
		t.Errorf("New() with numeric charset returned error: %v", err)
	}
	if len(id) != 10 {
		t.Errorf("New() with numeric charset returned ID of length %d, expected 10", len(id))
	}
	for _, char := range id {
		if !strings.ContainsRune(numericCharset, char) {
			t.Errorf("New() with numeric charset returned invalid character: %c", char)
		}
	}

	// Test with single character charset
	singleCharset := "X"
	id, err = New(5, singleCharset)
	if err != nil {
		t.Errorf("New() with single character charset returned error: %v", err)
	}
	if id != "XXXXX" {
		t.Errorf("New() with single character charset returned %s, expected XXXXX", id)
	}

	// Test with special characters charset
	specialCharset := "!@#$%"
	id, err = New(8, specialCharset)
	if err != nil {
		t.Errorf("New() with special characters charset returned error: %v", err)
	}
	if len(id) != 8 {
		t.Errorf("New() with special characters charset returned ID of length %d, expected 8", len(id))
	}
	for _, char := range id {
		if !strings.ContainsRune(specialCharset, char) {
			t.Errorf("New() with special characters charset returned invalid character: %c", char)
		}
	}

	// Test with zero length and custom charset
	id, err = New(0, "ABC")
	if err != nil {
		t.Errorf("New(0) with custom charset returned error: %v", err)
	}
	if id != "" {
		t.Errorf("New(0) with custom charset returned non-empty ID: %s", id)
	}
}

func TestNewMultiCustomCharset(t *testing.T) {
	// Test NewMulti with custom charset
	customCharset := "ABCDEF"
	ids, err := NewMulti(6, 3, customCharset)
	if err != nil {
		t.Errorf("NewMulti() with custom charset returned error: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("NewMulti() with custom charset returned slice of length %d, expected 3", len(ids))
	}

	for i, id := range ids {
		if len(id) != 6 {
			t.Errorf("NewMulti() ID %d has length %d, expected 6", i, len(id))
		}
		for _, char := range id {
			if !strings.ContainsRune(customCharset, char) {
				t.Errorf("NewMulti() ID %d contains invalid character: %c", i, char)
			}
		}
	}

	// Test with zero quantity and custom charset
	ids, err = NewMulti(5, 0, "XYZ")
	if err != nil {
		t.Errorf("NewMulti() with zero quantity and custom charset returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("NewMulti() with zero quantity returned non-empty slice: %v", ids)
	}
}
