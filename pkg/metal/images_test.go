package metal

import (
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestGetOsAndSemverFromImage(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		want       string
		wantSemver *semver.Version
		wantErr    error
	}{
		{
			name:       "shorthand syntax",
			id:         "ubuntu-19.04",
			want:       "ubuntu",
			wantSemver: semver.MustParse("19.04"),
		},
		{
			name:       "fqn syntax",
			id:         "ubuntu-19.04.20200408",
			want:       "ubuntu",
			wantSemver: semver.MustParse("19.04.20200408"),
		},
		{
			name:       "dashes in os variant",
			id:         "ubuntu-slim-19.04.20200408",
			want:       "ubuntu-slim",
			wantSemver: semver.MustParse("19.04.20200408"),
		},
		{
			name:    "no version contained",
			id:      "ubuntu",
			wantErr: fmt.Errorf("invalid format for os image, expected <os>-<major>.<minor>[.<patch>]: ubuntu"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, sem, err := GetOsAndSemverFromImage(tt.id)

			if diff := cmp.Diff(err, tt.wantErr, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(v, tt.want); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
			if diff := cmp.Diff(sem, tt.wantSemver); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
