package genericcli

import "math"

type Truncatable interface {
	~string
}

const TruncateEllipsis = "..."

// TruncateMiddle will trim a string in the middle.
func TruncateMiddle[T Truncatable](input T, maxlength int) T {
	return TruncateMiddleEllipsis(input, TruncateEllipsis, maxlength)
}

// TruncateMiddleEllipsis will trim a string in the middle and replace it with ellipsis.
func TruncateMiddleEllipsis[T Truncatable](input T, ellipsis T, maxlength int) T {
	if ellipsis == "" {
		ellipsis = TruncateEllipsis
	}
	if maxlength < 0 || len(input) <= maxlength {
		return input
	}

	finalLength := float64(maxlength - len(ellipsis))
	if finalLength <= 0 {
		return input[:maxlength]
	}

	var (
		start = int(math.Ceil(finalLength / 2))
		end   = int(math.Floor(finalLength / 2))
	)
	return input[:start] + ellipsis + input[len(input)-end:]
}

// TruncateEnd will trim a string at the end.
func TruncateEnd[T Truncatable](input T, maxlength int) T {
	return TruncateEndEllipsis(input, TruncateEllipsis, maxlength)
}

// TruncateEndEllipsis will trim a string at the end and replace it with ellipsis.
func TruncateEndEllipsis[T Truncatable](input T, ellipsis T, maxlength int) T {
	if ellipsis == "" {
		ellipsis = TruncateEllipsis
	}
	if maxlength < 0 || len(input) <= maxlength {
		return input
	}

	finalLength := maxlength - len(ellipsis)
	if finalLength <= 0 {
		return input[:maxlength]
	}

	return input[:finalLength] + ellipsis
}

// TruncateStart will trim a string at the start.
func TruncateStart[T Truncatable](input T, maxlength int) T {
	return TruncateStartEllipsis(input, TruncateEllipsis, maxlength)
}

// TruncateStartEllipsis will trim a string at the start and replace it with ellipsis.
func TruncateStartEllipsis[T Truncatable](input T, ellipsis T, maxlength int) T {
	if ellipsis == "" {
		ellipsis = TruncateEllipsis
	}
	if maxlength < 0 || len(input) <= maxlength {
		return input
	}

	finalLength := maxlength - len(ellipsis)
	if finalLength <= 0 {
		return input[len(input)-maxlength:]
	}

	return ellipsis + input[len(input)-finalLength:]
}
