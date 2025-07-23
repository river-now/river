package jsonutil

import (
	"encoding/json"
	"fmt"
)

type JSONString string

func Serialize(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("error encoding JSON: %w", err)
	}
	return data, nil
}

func Parse[T any](data []byte) (T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return v, fmt.Errorf("error decoding JSON: %w", err)
	}
	return v, nil
}
