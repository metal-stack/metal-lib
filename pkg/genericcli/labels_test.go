package genericcli

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestLabelsToMap(t *testing.T) {
	tests := []struct {
		name    string
		labels  []string
		want    map[string]string
		wantErr error
	}{
		{
			name:   "empty labels",
			labels: []string{},
			want:   map[string]string{},
		},
		{
			name:   "valid labels",
			labels: []string{"a=b", "c=d"},
			want: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		{
			name:   "you can set to empty string",
			labels: []string{"a="},
			want: map[string]string{
				"a": "",
			},
		},
		{
			name:    "invalid label",
			labels:  []string{"a=b", "c"},
			want:    nil,
			wantErr: errors.New("provided labels must be in the form <key>=<value>, found: c"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LabelsToMap(tt.labels)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
