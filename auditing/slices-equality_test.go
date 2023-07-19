package auditing

import (
	"fmt"
	"testing"
)

func TestSlicesUnorderedEqual(t *testing.T) {
	tt := []struct {
		expect    bool
		ordered   []int
		unordered []int
	}{
		{true, nil, nil},
		{true, nil, []int{}},
		{true, []int{}, []int{}},
		{false, []int{0}, []int{1}},
		{false, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, []int{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{true, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{true, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, []int{5, 4, 9, 8, 1, 0, 3, 7, 6, 2}},
		{false, []int{1, 2, 3}, []int{7, 6, 5}},
		{false, []int{1, 1, 1}, []int{1, 1}},
		{true, []int{1, 2, 1, 2}, []int{2, 1, 1, 2}},
		{false, []int{1, 1, 1, 2}, []int{2, 2, 2, 1}},
	}

	for i, run := range tt {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			got := slicesUnorderedEqual(run.ordered, run.unordered)
			if run.expect && got != run.expect {
				t.Errorf("expected equal elements, %v %v", run.ordered, run.unordered)
			} else if got != run.expect {
				t.Errorf("expected distinct elements, %v %v", run.ordered, run.unordered)
			}
		})
	}
}
