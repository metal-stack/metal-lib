package pointer

import "reflect"

// Pointer returns the pointer of the given value.
func Pointer[T any](t T) *T {
	return &t
}

// PointerOrDefault returns the pointer of the given value.
// If the given value is equal to the zero value, the pointer of the default value will be returned instead.
func PointerOrDefault[T any](t T, defaultValue T) *T {
	if IsZero(t) {
		return Pointer(defaultValue)
	}

	return Pointer(t)
}

// SafeDeref returns the value from the passed pointer or zero value for a nil pointer.
func SafeDeref[T any](t *T) T {
	if t == nil {
		var zero T

		return zero
	}

	return *t
}

// SafeDerefOrDefault returns the value from the passed pointer or the default value for a nil pointer or zero value.
func SafeDerefOrDefault[T any](t *T, defaultValue T) T {
	if t == nil {
		return defaultValue
	}

	if IsZero(*t) {
		return defaultValue
	}

	return *t
}

// IsZero returns true if the passed value equals its zero value.
func IsZero[T any](t T) bool {
	var zero T
	return reflect.DeepEqual(t, zero)
}
