package genericcli

import "math"

type Truncatable interface {
	~string
}

const TruncateElipsis = "..."

// TruncateMiddle will trim a string in the middle.
func TruncateMiddle[T Truncatable](input T, maxlength int) T {
	return TruncateMiddleElipsis(input, TruncateElipsis, maxlength)
}

// TruncateMiddleElipsis will trim a string in the middle and replace it with elipsis.
func TruncateMiddleElipsis[T Truncatable](input T, elipsis T, maxlength int) T {
	if elipsis == "" {
		elipsis = TruncateElipsis
	}
	if maxlength < 0 || len(input) <= maxlength {
		return input
	}

	finalLength := float64(maxlength - len(elipsis))
	if finalLength <= 0 {
		return input[:maxlength]
	}

	var (
		start = int(math.Ceil(finalLength / 2))
		end   = int(math.Floor(finalLength / 2))
	)
	return input[:start] + elipsis + input[len(input)-end:]
}

// TruncateEnd will trim a string at the end.
func TruncateEnd[T Truncatable](input T, maxlength int) T {
	return TruncateEndElipsis(input, TruncateElipsis, maxlength)
}

// TruncateEndElipsis will trim a string at the end and replace it with elipsis.
func TruncateEndElipsis[T Truncatable](input T, elipsis T, maxlength int) T {
	if elipsis == "" {
		elipsis = TruncateElipsis
	}
	if maxlength < 0 || len(input) <= maxlength {
		return input
	}

	finalLength := maxlength - len(elipsis)
	if finalLength <= 0 {
		return input[:maxlength]
	}

	return input[:finalLength] + elipsis
}

// TruncateStart will trim a string at the start.
func TruncateStart[T Truncatable](input T, maxlength int) T {
	return TruncateStartElipsis(input, TruncateElipsis, maxlength)
}

// TruncateStartElipsis will trim a string at the start and replace it with elipsis.
func TruncateStartElipsis[T Truncatable](input T, elipsis T, maxlength int) T {
	if elipsis == "" {
		elipsis = TruncateElipsis
	}
	if maxlength < 0 || len(input) <= maxlength {
		return input
	}

	finalLength := maxlength - len(elipsis)
	if finalLength <= 0 {
		return input[len(input)-maxlength:]
	}

	return elipsis + input[len(input)-finalLength:]
}
