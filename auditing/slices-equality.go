package auditing

func slicesUnorderedEqual[T comparable](lhs, rhs []T) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	lvals := make(map[T]int, len(lhs))
	for _, l := range lhs {
		lvals[l]++
	}
	for _, r := range rhs {
		if lvals[r] == 0 {
			return false
		}
		lvals[r]--
	}
	return true
}
