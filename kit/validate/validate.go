// Package validate provides a simple way to validate and parse data from HTTP requests.
package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

// JSONBodyInto decodes an HTTP request body into a struct and validates it.
func JSONBodyInto(r *http.Request, destStructPtr any) error {
	if err := json.NewDecoder(r.Body).Decode(destStructPtr); err != nil {
		return &ValidationError{Err: fmt.Errorf("error decoding JSON: %w", err)}
	}
	if err := attemptValidation("validate.JSONBodyInto", destStructPtr); err != nil {
		return err
	}
	return nil
}

// JSONBytesInto decodes a byte slice containing JSON data into a struct and validates it.
func JSONBytesInto(data []byte, destStructPtr any) error {
	if err := json.Unmarshal(data, destStructPtr); err != nil {
		return &ValidationError{Err: fmt.Errorf("error decoding JSON: %w", err)}
	}
	if err := attemptValidation("validate.JSONBytesInto", destStructPtr); err != nil {
		return err
	}
	return nil
}

// JSONStrInto decodes a string containing JSON data into a struct and validates it.
func JSONStrInto(data string, destStructPtr any) error {
	if err := json.Unmarshal([]byte(data), destStructPtr); err != nil {
		return &ValidationError{Err: fmt.Errorf("error decoding JSON: %w", err)}
	}
	if err := attemptValidation("validate.JSONStrInto", destStructPtr); err != nil {
		return err
	}
	return nil
}

// URLSearchParamsInto parses the URL parameters of an HTTP request into a struct and validates it.
func URLSearchParamsInto(r *http.Request, destStructPtr any) error {
	if err := parseURLValues(r.URL.Query(), destStructPtr); err != nil {
		return &ValidationError{Err: fmt.Errorf("error parsing URL parameters: %w", err)}
	}
	if err := attemptValidation("validate.URLSearchParamsInto", destStructPtr); err != nil {
		return err
	}
	return nil
}

func attemptValidation(label string, x any) error {
	if errs := safeRunOwnValidate(label, x, getTypeState(reflect.ValueOf(x))); len(errs) > 0 {
		return &ValidationError{Err: errors.Join(errs...)}
	}
	return nil
}
