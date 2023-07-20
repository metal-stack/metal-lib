package k8s

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestLessThan(t *testing.T) {
	tests := []struct {
		name    string
		actual  string
		target  *semver.Version
		want    bool
		wantErr bool
	}{
		{
			name:    "1.18",
			actual:  "1.18.5",
			target:  KubernetesV119,
			want:    true,
			wantErr: false,
		},
		{
			name:    "1.19",
			actual:  "1.19.5",
			target:  KubernetesV119,
			want:    false,
			wantErr: false,
		},
		{
			name:    "1.19",
			actual:  "1.19.0",
			target:  KubernetesV119,
			want:    false,
			wantErr: false,
		},
		{
			name:    "1.20",
			actual:  "v1.20.5",
			target:  KubernetesV119,
			want:    false,
			wantErr: false,
		},
		{
			name:    "wrong version",
			actual:  "ab.1.c",
			target:  KubernetesV119,
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LessThan(tt.actual, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("LessThan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LessThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGreaterThan(t *testing.T) {
	tests := []struct {
		name    string
		actual  string
		target  *semver.Version
		want    bool
		wantErr bool
	}{
		{
			name:    "1.18",
			actual:  "1.18.5",
			target:  KubernetesV119,
			want:    false,
			wantErr: false,
		},
		{
			name:    "1.19",
			actual:  "1.19.5",
			target:  KubernetesV119,
			want:    true,
			wantErr: false,
		},
		{
			name:    "1.20",
			actual:  "v1.20.5",
			target:  KubernetesV119,
			want:    true,
			wantErr: false,
		},
		{
			name:    "1.20",
			actual:  "v1.20.0",
			target:  KubernetesV119,
			want:    true,
			wantErr: false,
		},
		{
			name:    "1.19",
			actual:  "1.19.0",
			target:  KubernetesV119,
			want:    true,
			wantErr: false,
		},
		{
			name:    "wrong version",
			actual:  "ab.1.c",
			target:  KubernetesV119,
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GreaterThanOrEqual(tt.actual, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("GreaterThan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GreaterThan() = %v, want %v", got, tt.want)
			}
		})
	}
}
