package multisort

import "cmp"

// CompareFn is compare function that compares the two given values according to multisort criteria.
//
// The function is implemented for comparable data types inside this package. Use it!
//
// It returns:
// - Less when a is smaller than b (on descending: b is smaller than a)
// - NotEqual when a is not equal to b
// - 0 when neither less or not equal is returned
type CompareFn[E any] func(a E, b E, descending bool) CompareResult

type CompareResult int

const (
	Less     CompareResult = 1
	NotEqual CompareResult = 2
)

// Compare compares values according to multisort criteria.
func Compare[O cmp.Ordered](a O, b O, descending bool) CompareResult {
	if descending {
		if b < a {
			return Less
		}
	} else {
		if a < b {
			return Less
		}
	}

	if a != b {
		return NotEqual
	}

	return 0
}

// WithCompareFunc compares input and returns a CompareResult according to multisort criteria.
func WithCompareFunc(f func() int, descending bool) CompareResult {
	r := f()

	if descending {
		if r == 1 {
			return Less
		}
	} else {
		if r == -1 {
			return Less
		}
	}

	if r != 0 {
		return NotEqual
	}

	return 0
}
