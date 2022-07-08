package pointer

import "reflect"

// Pointer returns the pointer of the given value.
func Pointer[T any](t T) *T {
	return &t
}

// PointerOrDefault returns the pointer of the given value.
// If the given value is equal to the zero value, the pointer of the default value will be returned instead.
func PointerOrDefault[T any](t T, defaultValue T) *T {
	var zero T

	if reflect.DeepEqual(t, zero) {
		return Pointer(defaultValue)
	}

	return Pointer(t)
}

// Deref returns the value from the passed pointer or zero value for a nil pointer.
func Deref[T any](t *T) T { //nolint:ireturn
	if t == nil {
		var zero T

		return zero
	}

	return *t
}

// DerefOrDefault returns the value from the passed pointer or the default value for a nil pointer or zero value.
func DerefOrDefault[T any](t *T, defaultValue T) T { //nolint:ireturn
	if t == nil {
		return defaultValue
	}

	var zero T

	if reflect.DeepEqual(*t, zero) {
		return defaultValue
	}

	return *t
}
