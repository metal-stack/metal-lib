package pointer

// FirstOrZero returns the first value of a slice or the zero slice if slize is empty.
func FirstOrZero[T any](t []T) T {
	if len(t) == 0 {
		var zero T

		return zero
	}

	return t[0]
}

// FirstOrZero returns a slice that wraps the given value.
func WrapInSlice[T any](t T) []T {
	return []T{t}
}
