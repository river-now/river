package validate

import (
	"errors"
	"fmt"
	"reflect"
)

type Validator interface{ Validate() error }

type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

/////////////////////////////////////////////////////////////////////
/////// ANY CHECKER
/////////////////////////////////////////////////////////////////////

type AnyChecker struct {
	label            string
	trueValue        any
	baseReflectValue reflect.Value
	typeState

	done   bool
	errors []error
}

func newAnyChecker(label string, trueValue any, reflectValue reflect.Value) *AnyChecker {
	return &AnyChecker{
		label:            label,
		trueValue:        trueValue,
		baseReflectValue: safeDereference(reflectValue),
		typeState:        getTypeState(reflectValue),
	}
}

func (c *AnyChecker) Required() *AnyChecker { return c.init(true) }
func (c *AnyChecker) Optional() *AnyChecker { return c.init(false) }

func (c *AnyChecker) Error() error {
	if len(c.errors) > 0 {
		return &ValidationError{Err: errors.Join(c.errors...)}
	}
	return nil
}

func (c *AnyChecker) ok() { c.done = true }

func (c *AnyChecker) fail(errMsg string) {
	c.done = true
	c.errors = append(c.errors, errors.New(errMsg))
}

func (c *AnyChecker) failF(format string, args ...any) {
	c.fail(fmt.Sprintf(format, args...))
}

func (c *AnyChecker) init(required bool) *AnyChecker {
	if c.done {
		return c
	}
	if c.trueValue == nil || isEffectivelyZero(c.reflectValue) {
		if required {
			c.fail(fmt.Sprintf("%s is required", c.label))
		} else {
			c.ok()
		}
		return c
	}
	c.safeRunOwnValidate()
	return c
}

func (c *AnyChecker) safeRunOwnValidate() {
	if errors := safeRunOwnValidate(c.label, c.trueValue, c.typeState); len(errors) > 0 {
		c.errors = append(c.errors, errors...)
	}
}

/////////////////////////////////////////////////////////////////////
/////// OBJECT CHECKER
/////////////////////////////////////////////////////////////////////

type ObjectChecker struct {
	AnyChecker
	ChildCheckers []*AnyChecker
}

func (oc *ObjectChecker) Required(field string) *AnyChecker { return oc.validateField(field, true) }
func (oc *ObjectChecker) Optional(field string) *AnyChecker { return oc.validateField(field, false) }

func (oc *ObjectChecker) Error() error {
	for _, child := range oc.ChildCheckers {
		if err := child.Error(); err != nil {
			oc.errors = append(oc.errors, err)
		}
	}
	if len(oc.errors) > 0 {
		return &ValidationError{Err: errors.Join(oc.errors...)}
	}
	return nil
}

func (oc *ObjectChecker) validateField(fieldName string, required bool) (c *AnyChecker) {
	if oc.done {
		c = newAnyChecker(fieldName, nil, reflect.Value{})
		c.done = true
		return c
	}
	wrappedField := oc.getFieldValue(fieldName)
	c = newAnyChecker(fieldName, wrappedField.trueValue, wrappedField.reflectValue)
	oc.ChildCheckers = append(oc.ChildCheckers, c)
	if required {
		c.Required()
	} else {
		c.Optional()
	}
	return
}

func (oc *ObjectChecker) getFieldValue(fieldName string) (wrapped *fieldWrapper) {
	wrapped = &fieldWrapper{}
	if oc.isMapWithStrKeysLike {
		key := reflect.ValueOf(fieldName)
		wrapped.reflectValue = oc.baseReflectValue.MapIndex(key)
		if !wrapped.reflectValue.IsValid() {
			return
		}
		wrapped.trueValue = wrapped.reflectValue.Interface()
		return
	}
	if oc.isStructLike {
		wrapped.reflectValue = oc.baseReflectValue.FieldByName(fieldName)
		if !wrapped.reflectValue.IsValid() || !wrapped.reflectValue.CanInterface() {
			return
		}
		wrapped.trueValue = wrapped.reflectValue.Interface()
		return
	}
	panic("this should never happen")
}

/////////////////////////////////////////////////////////////////////
/////// CORE ENTRY POINTS
/////////////////////////////////////////////////////////////////////

// An "object" as defined by this library is either (1) a struct,
// (2) a map with string keys, or (3) a pointer to (1) or (2). If
// you want to add field-level validation rules to an object, use
// this entry point. If you are not validating an object, or you
// just want any embedded fields that implement Validator to be
// validated, you can use the Any function. If the target is an
// object, both Object() and Any() will auto-validate any of the
// object's fields that implement Validator.

func Any(label string, anything any) *AnyChecker {
	return newAnyChecker(label, anything, reflect.ValueOf(anything))
}

func Object(object any) *ObjectChecker {
	oc := &ObjectChecker{}
	if object == nil {
		oc.fail("object cannot be nil")
		return oc
	}
	reflectValue := reflect.ValueOf(object)
	typeState := getTypeState(reflectValue)
	if !typeState.isStructLike && !typeState.isMapWithStrKeysLike {
		oc.failF("object must be a struct or a map with string keys (got %T)", object)
		return oc
	}
	oc.label = reflectValue.Type().String()
	oc.trueValue = object
	oc.reflectValue = reflectValue
	oc.baseReflectValue = safeDereference(reflectValue)
	oc.typeState = typeState
	return oc
}

/////////////////////////////////////////////////////////////////////
/////// UTILS
/////////////////////////////////////////////////////////////////////

func safeRunOwnValidate(label string, trueValue any, typeState typeState) []error {
	var errors []error

	if safeIsNil(typeState.reflectValue) {
		return errors
	}

	if impl, ok := trueValue.(Validator); ok {
		if err := impl.Validate(); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", label, err))
		}
	}
	if typeState.isStructLike || typeState.isMapLike {
		if locErrs := callValidateOnStructOrMapElements(label, typeState); len(locErrs) > 0 {
			errors = append(errors, locErrs...)
		}
	}
	if typeState.isSliceOrArrayLike {
		if locErrs := callValidateOnSliceOrArrayElements(label, typeState.reflectValue); len(locErrs) > 0 {
			errors = append(errors, locErrs...)
		}
	}
	return errors
}

func callValidateOnStructOrMapElements(label string, typeState typeState) []error {
	var errors []error
	reflectValue := typeState.reflectValue
	if typeState.isMapWithStrKeysLike {
		base := safeDereference(reflectValue)
		for _, keyReflectValue := range base.MapKeys() {
			// KEY
			keyLabel := fmt.Sprintf("%s[%s]", label, keyReflectValue.String())
			keyTrueValue := keyReflectValue.Interface()
			keyTypeState := getTypeState(keyReflectValue)
			if locErrs := safeRunOwnValidate(keyLabel, keyTrueValue, keyTypeState); len(locErrs) > 0 {
				errors = append(errors, locErrs...)
			}
			// VALUE
			valReflectValue := base.MapIndex(keyReflectValue)
			valLabel := fmt.Sprintf("%s[%s]", label, keyReflectValue.String())
			valTrueValue := valReflectValue.Interface()
			valTypeState := getTypeState(valReflectValue)
			if locErrs := safeRunOwnValidate(valLabel, valTrueValue, valTypeState); len(locErrs) > 0 {
				errors = append(errors, locErrs...)
			}
		}
	} else if typeState.isStructLike {
		base := safeDereference(reflectValue)
		for i := range base.NumField() {
			fieldReflectValue := base.Field(i)
			if !fieldReflectValue.CanInterface() {
				continue
			}
			label := fmt.Sprintf("%s.%s", label, base.Type().Field(i).Name)
			trueValue := fieldReflectValue.Interface()
			typeState := getTypeState(fieldReflectValue)
			if locErrs := safeRunOwnValidate(label, trueValue, typeState); len(locErrs) > 0 {
				errors = append(errors, locErrs...)
			}
		}
	}
	return errors
}

func callValidateOnSliceOrArrayElements(label string, reflectValue reflect.Value) []error {
	var errors []error
	base := safeDereference(reflectValue)
	for i := range base.Len() {
		elReflectValue := base.Index(i)
		elLabel := fmt.Sprintf("%s[%d]", label, i)
		elTrueValue := elReflectValue.Interface()
		elTypeState := getTypeState(elReflectValue)
		if locErrors := safeRunOwnValidate(elLabel, elTrueValue, elTypeState); len(locErrors) > 0 {
			errors = append(errors, locErrors...)
		}
	}
	return errors
}

func safeDereference(reflectValue reflect.Value) reflect.Value {
	if reflectValue.Kind() == reflect.Ptr {
		return reflectValue.Elem()
	}
	return reflectValue
}

type typeState struct {
	reflectValue         reflect.Value
	isStructLike         bool
	isMapLike            bool
	isMapWithStrKeysLike bool
	isSliceOrArrayLike   bool
}

func getTypeState(reflectValue reflect.Value) typeState {
	base := safeDereference(reflectValue)
	isMapLike := base.Kind() == reflect.Map
	isMapWithStrKeysLike := isMapLike && base.Type().Key().Kind() == reflect.String
	return typeState{
		reflectValue:         reflectValue,
		isStructLike:         base.Kind() == reflect.Struct,
		isMapLike:            isMapLike,
		isMapWithStrKeysLike: isMapWithStrKeysLike,
		isSliceOrArrayLike:   base.Kind() == reflect.Slice || base.Kind() == reflect.Array,
	}
}

type fieldWrapper struct {
	trueValue    any
	reflectValue reflect.Value
}

func (fw *fieldWrapper) isTruthy() bool {
	return !isEffectivelyZero(fw.reflectValue)
}

func isEffectivelyZero(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return true
		}
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
	}
	switch v.Kind() {
	case reflect.Struct:
		return false
	case reflect.Map, reflect.Slice:
		return v.IsNil()
	default:
		return v.IsZero()
	}
}

func safeIsNil(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		return v.IsNil()
	}
	return false
}

func attemptValidation(label string, x any) error {
	if errs := safeRunOwnValidate(label, x, getTypeState(reflect.ValueOf(x))); len(errs) > 0 {
		return &ValidationError{Err: errors.Join(errs...)}
	}
	return nil
}
