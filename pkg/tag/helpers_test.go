package tag

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTagMap_Contains(t *testing.T) {
	tests := []struct {
		name   string
		labels []string
		tag    string
		value  string
		want   bool
	}{
		{
			name:   "empty map",
			labels: nil,
			tag:    ClusterID,
			value:  "test",
			want:   false,
		},
		{
			name: "not contains",
			labels: []string{
				fmt.Sprintf("%s=%s", ClusterEgress, "1.2.3.4"),
				fmt.Sprintf("%s=%s", ClusterName, "test cluster"),
			},
			tag:   ClusterID,
			value: "test",
			want:  false,
		},
		{
			name: "contains label",
			labels: []string{
				"label-with-no-assignment",
				fmt.Sprintf("%s=%s", ClusterEgress, "1.2.3.4"),
				fmt.Sprintf("%s=%s", ClusterID, "test"),
				fmt.Sprintf("%s=%s", ClusterName, "test cluster"),
			},
			tag:   ClusterID,
			value: "test",
			want:  true,
		},
		{
			name: "contains label with no assignment",
			labels: []string{
				"label-with-no-assignment",
				fmt.Sprintf("%s=%s", ClusterEgress, "1.2.3.4"),
				fmt.Sprintf("%s=%s", ClusterID, "test"),
				fmt.Sprintf("%s=%s", ClusterName, "test cluster"),
			},
			tag:   "label-with-no-assignment",
			value: "",
			want:  true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMap(tt.labels)
			if got := tm.Contains(tt.tag, tt.value); got != tt.want {
				t.Errorf("TagMap.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagMap_Value(t *testing.T) {
	tests := []struct {
		name      string
		labels    []string
		tag       string
		want      bool
		wantValue string
	}{
		{
			name:      "empty map",
			labels:    nil,
			tag:       ClusterID,
			want:      false,
			wantValue: "",
		},
		{
			name: "get value",
			labels: []string{
				"label-with-no-assignment",
				fmt.Sprintf("%s=%s", ClusterEgress, "1.2.3.4"),
				fmt.Sprintf("%s=%s", ClusterID, "test"),
				fmt.Sprintf("%s=%s", ClusterName, "test cluster"),
			},
			tag:       ClusterID,
			want:      true,
			wantValue: "test",
		},
		{
			name: "not contained",
			labels: []string{
				"label-with-no-assignment",
				fmt.Sprintf("%s=%s", ClusterEgress, "1.2.3.4"),
				fmt.Sprintf("%s=%s", ClusterName, "test cluster"),
			},
			tag:       ClusterID,
			want:      false,
			wantValue: "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tm := NewTagMap(tt.labels)
			gotValue, got := tm.Value(tt.tag)
			if got != tt.want {
				t.Errorf("TagMap.Value() = %v, want %v", got, tt.want)
			}
			if diff := cmp.Diff(gotValue, tt.wantValue); diff != "" {
				t.Errorf("TagMap.Value() diff = %s", diff)
			}
		})
	}
}
