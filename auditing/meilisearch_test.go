package auditing

import (
	"fmt"
	"testing"
	"time"
)

func TestMeilisearchRelevantIndexNames(t *testing.T) {
	testCases := []struct {
		indexName  string
		isRelevant bool
		from       string
		to         string
	}{
		{"metal-2023-07-26", true, "2023-07-05 12:07", "2023-08-01 0:01"},
		{"metal-stack-2023-07-26", true, "2023-07-05 12:07", "2023-08-01 0:01"},
		{"cloud-2023-07-26", true, "2023-07-26 12:07", "2023-07-27 0:01"},

		{"metal-2023-07-26", false, "2023-08-05 12:07", "2023-09-01 0:01"},
		{"metal-stack-2023-07-26", false, "2022-07-05 12:07", "2022-08-01 0:01"},
		{"cloud-2023-07-26", false, "2023-07-27 0:00", "2023-08-01 0:01"},

		{"metal-2023-07", true, "2023-07-05 12:07", "2023-08-01 0:01"},
		{"metal-stack-2023-07", true, "2023-07-05 12:07", "2023-08-01 0:01"},
		{"cloud-2023-07-26", true, "2023-07-26 12:07", "2023-07-27 0:01"},

		{"metal-2023-07", false, "2023-08-05 12:07", "2023-09-01 0:01"},
		{"metal-stack-2023-07", false, "2022-07-05 12:07", "2022-08-01 0:01"},
		{"cloud-2023-07", false, "2023-06-27 0:00", "2023-06-29 0:01"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d is %s relevant?", i, tc.indexName), func(t *testing.T) {
			format := "2006-01-02 3:04"
			from, err := time.Parse(format, tc.from)
			if err != nil {
				t.Error(err)
			}
			to, err := time.Parse(format, tc.to)
			if err != nil {
				t.Error(err)
			}

			got := isIndexRelevantForSearchRange(tc.indexName, from, to)
			if got != tc.isRelevant {
				t.Errorf("got %t, want %t", got, tc.isRelevant)
			}
		})
	}
}
