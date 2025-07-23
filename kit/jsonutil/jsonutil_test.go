package jsonutil

import (
	"testing"
)

func TestSerializeAndParse(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	original := TestStruct{Name: "Alice", Age: 25}

	// Serialize the struct to JSON
	jsonData, err := Serialize(original)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Parse the JSON back into a struct
	var parsed TestStruct
	parsed, err = Parse[TestStruct](jsonData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if original != parsed {
		t.Fatalf("expected parsed struct to match original struct")
	}

	// Test parsing with invalid JSON
	invalidJSON := []byte(`{"name": "Bob", "age": "twenty"}`) // Invalid because age is not an int
	_, err = Parse[TestStruct](invalidJSON)
	if err == nil {
		t.Fatalf("expected error for invalid JSON, got nil")
	}

	// Test parsing with empty JSON
	emptyJSON := []byte(``)
	_, err = Parse[TestStruct](emptyJSON)
	if err == nil {
		t.Fatalf("expected error for empty JSON, got nil")
	}
}
