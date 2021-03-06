package testutil

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// Ceq is a custom equals check; the given function returns true if its arguments are equal
func Ceq(t *testing.T, expected, actual interface{}, eq func(a, b interface{}) bool, context ...interface{}) bool {
	t.Helper()
	e := eq(expected, actual)
	Require(t, e, mergeContext(context, "Expecting %v (%v), got %v (%v)", expected, reflect.TypeOf(expected), actual, reflect.TypeOf(actual))...)
	return e
}

// Cneq is a custom not-equals check; the given function returns true if its arguments are equal
func Cneq(t *testing.T, unexpected, actual interface{}, eq func(a, b interface{}) bool, context ...interface{}) bool {
	t.Helper()
	ne := !eq(unexpected, actual)
	Require(t, ne, mergeContext(context, "Value should not be %v (%v)", unexpected, reflect.TypeOf(unexpected))...)
	return ne
}

// Require is an assertion that logs a failure if its given argument is not true
func Require(t *testing.T, condition bool, context ...interface{}) {
	t.Helper()
	if !condition {
		if len(context) == 0 {
			t.Fatalf("Assertion failed")
		} else {
			msg := context[0].(string)
			// if any args were deferred (e.g. a function instead of a value), get those args now
			args := make([]interface{}, len(context)-1)
			for i, a := range context[1:] {
				rv := reflect.ValueOf(a)
				if rv.Kind() == reflect.Func {
					a = rv.Call([]reflect.Value{})[0].Interface()
				}
				args[i] = a
			}
			t.Fatalf(msg, args...)
		}
	}
}

func mergeContext(context []interface{}, msg string, msgArgs ...interface{}) []interface{} {
	if len(context) == 0 {
		ret := make([]interface{}, 0, len(msgArgs)+1)
		ret = append(ret, msg)
		ret = append(ret, msgArgs...)
		return ret
	} else {
		ret := make([]interface{}, 0, len(context)+len(msgArgs))
		ret = append(ret, msg+": "+context[0].(string))
		ret = append(ret, msgArgs...)
		ret = append(ret, context[1:]...)
		return ret
	}
}

// Ok asserts that the given error is nil
func Ok(t *testing.T, err error, context ...interface{}) {
	t.Helper()
	Require(t, err == nil, mergeContext(context, "Unexpected error: %s", func() interface{} { return err.Error() })...)
}

// Nok asserts that the given error is not nil
func Nok(t *testing.T, err error, context ...interface{}) {
	t.Helper()
	Require(t, err != nil, mergeContext(context, "Expected error but got none")...)
}

// Eq asserts that the given two values are equal
func Eq(t *testing.T, expected, actual interface{}, context ...interface{}) bool {
	t.Helper()
	return Ceq(t, expected, actual, eqany, context...)
}

// Neq asserts that the given two values are not equal
func Neq(t *testing.T, unexpected, actual interface{}, context ...interface{}) bool {
	t.Helper()
	return Cneq(t, unexpected, actual, eqany, context...)
}

// default equality test and helpers

func eqany(expected, actual interface{}) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}

	// We don't want reflect.DeepEquals because of its recursive nature. So we need
	// a custom compare for slices and maps. Two slices are equal if they have the
	// same number of elements and the elements at the same index are equal to each
	// other. Two maps are equal if their key sets are the same and the corresponding
	// values are equal. (The relationship is not recursive,  slices or maps that
	// contain other slices or maps can't be tested.)
	et := reflect.TypeOf(expected)

	if et.Kind() == reflect.Slice {
		return eqslice(reflect.ValueOf(expected), reflect.ValueOf(actual))
	} else if et.Kind() == reflect.Map {
		return eqmap(reflect.ValueOf(expected), reflect.ValueOf(actual))
	} else {
		return eqscalar(expected, actual)
	}
}

func eqscalar(expected, actual interface{}) bool {
	// special-case simple equality for []byte (since slices aren't directly comparable)
	if e, ok := expected.([]byte); ok {
		a, ok := actual.([]byte)
		return ok && bytes.Equal(e, a)
	}
	// and special-cases to handle NaN
	if e, ok := expected.(float32); ok && math.IsNaN(float64(e)) {
		a, ok := actual.(float32)
		return ok && math.IsNaN(float64(a))
	}
	if e, ok := expected.(float64); ok && math.IsNaN(e) {
		a, ok := actual.(float64)
		return ok && math.IsNaN(a)
	}
	// simple logic for everything else
	return expected == actual
}

func eqslice(expected, actual reflect.Value) bool {
	if expected.Len() != actual.Len() {
		return false
	}
	for i := 0; i < expected.Len(); i++ {
		e := expected.Index(i).Interface()
		a := actual.Index(i).Interface()
		if !eqscalar(e, a) {
			return false
		}
	}
	return true
}

func eqmap(expected, actual reflect.Value) bool {
	if expected.Len() != actual.Len() {
		return false
	}
	for _, k := range expected.MapKeys() {
		e := expected.MapIndex(k)
		a := actual.MapIndex(k)
		if !a.IsValid() {
			return false
		}
		if !eqscalar(e.Interface(), a.Interface()) {
			return false
		}
	}
	return true
}
