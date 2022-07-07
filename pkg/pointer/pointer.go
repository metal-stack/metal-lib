package pointer

// To returns the pointer of the given value.
func To[T any](t T) *T {
	return &t
}

// SafeDeref returns the value from the passed pointer or zero value for a nil pointer.
func SafeDeref[T any](t *T) T { //nolint:ireturn
	if t == nil {
		var zero T

		return zero
	}

	return *t
}
