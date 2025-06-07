// Package genericsutil provides helpers for implementing type erasure patterns
package genericsutil

import "errors"

/////////////////////////////////////////////////////////////////////
/////// ZERO HELPERS
/////////////////////////////////////////////////////////////////////

type AnyZeroHelper interface {
	I() any    // returns direct I zero val
	O() any    // returns direct O zero val
	IPtr() any // returns `new(I)` (pointer to I)
	OPtr() any // returns `new(O)` (pointer to O)
}

type ZeroHelper[I any, O any] struct{}

func (ZeroHelper[I, O]) I() any    { return Zero[I]() }
func (ZeroHelper[I, O]) O() any    { return Zero[O]() }
func (ZeroHelper[I, O]) IPtr() any { return new(I) }
func (ZeroHelper[I, O]) OPtr() any { return new(O) }

/////////////////////////////////////////////////////////////////////
/////// INPUT-OUTPUT FUNCTIONS
/////////////////////////////////////////////////////////////////////

type AnyIOFunc interface {
	AnyZeroHelper

	// If type assertion fails, implementation should still call the
	// underlying function with the zero value of the input type.
	ExecuteLoose(any) (any, error)

	// If type assertion fails, implementation should return an error
	// and the zero value of the input type, without calling the
	// underlying function.
	ExecuteStrict(any) (any, error)
}

type IOFunc[I any, O any] func(I) (O, error)

func (IOFunc[I, O]) I() any    { return Zero[I]() }
func (IOFunc[I, O]) O() any    { return Zero[O]() }
func (IOFunc[I, O]) IPtr() any { return new(I) }
func (IOFunc[I, O]) OPtr() any { return new(O) }

// If type assertion fails, ExecuteLoose will still call the underlying
// function with the zero value of the input type.
func (ioFunc IOFunc[I, O]) ExecuteLoose(input any) (any, error) {
	return ioFunc(AssertOrZero[I](input))
}

// If type assertion fails, ExecuteStrict will return an error and the
// zero value of the input type, without calling the underlying function.
func (ioFunc IOFunc[I, O]) ExecuteStrict(input any) (any, error) {
	var typedI, ok = input.(I)
	if !ok {
		return Zero[I](), errors.New("input type assertion failed")
	}
	return ioFunc(typedI)
}

/////////////////////////////////////////////////////////////////////
/////// UTILITIES
/////////////////////////////////////////////////////////////////////

// Simple alias for an empty struct
type None = struct{}

// Returns true if v is either an empty struct or a pointer to an empty struct
func IsNone(v any) bool {
	_, ok := v.(struct{})
	if ok {
		return true
	}
	_, ok = v.(*struct{})
	return ok
}

// Returns the zero value of type T
func Zero[T any]() T {
	var zero T
	return zero
}

// Returns v cast as type T if possible, otherwise returns the zero value of T
func AssertOrZero[T any](v any) T {
	if typedV, ok := v.(T); ok {
		return typedV
	}
	return Zero[T]()
}
