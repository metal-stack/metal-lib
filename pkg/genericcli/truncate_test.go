package genericcli

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTruncateMiddleEllipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		ellipsis  string
		maxlength int
		want      string
	}{
		{
			name:      "no trunc on short enough input",
			input:     "0123456789",
			ellipsis:  TruncateEllipsis,
			maxlength: 10,
			want:      "0123456789",
		},
		{
			name:      "even ellipsis is in the middle on even length",
			input:     "0123456789",
			ellipsis:  "..",
			maxlength: 6,
			want:      "01..89",
		},
		{
			name:      "even ellipsis is slightly to the right on odd length",
			input:     "0123456789",
			ellipsis:  "..",
			maxlength: 7,
			want:      "012..89",
		},
		{
			name:      "odd ellipsis is in the middle on odd length",
			input:     "0123456789",
			ellipsis:  TruncateEllipsis,
			maxlength: 7,
			want:      "01...89",
		},
		{
			name:      "odd ellipsis is slightly on the right on even length",
			input:     "0123456789",
			ellipsis:  TruncateEllipsis,
			maxlength: 6,
			want:      "01...9",
		},
		{
			name:      "too long ellipsis does not increase final length",
			input:     "0123456789",
			ellipsis:  "..........",
			maxlength: 6,
			want:      "012345",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateMiddleEllipsis(tt.input, tt.ellipsis, tt.maxlength)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestTruncateEndellipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		ellipsis  string
		maxlength int
		want      string
	}{
		{
			name:      "no trunc on short enough input",
			input:     "0123456789",
			ellipsis:  TruncateEllipsis,
			maxlength: 10,
			want:      "0123456789",
		},
		{
			name:      "even ellipsis",
			input:     "0123456789",
			ellipsis:  "..",
			maxlength: 6,
			want:      "0123..",
		},
		{
			name:      "odd ellipsis",
			input:     "0123456789",
			ellipsis:  "...",
			maxlength: 7,
			want:      "0123...",
		},
		{
			name:      "too long ellipsis does not increase final length",
			input:     "0123456789",
			ellipsis:  "..........",
			maxlength: 6,
			want:      "012345",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateEndEllipsis(tt.input, tt.ellipsis, tt.maxlength)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestTruncateStartellipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		ellipsis  string
		maxlength int
		want      string
	}{
		{
			name:      "no trunc on short enough input",
			input:     "0123456789",
			ellipsis:  TruncateEllipsis,
			maxlength: 10,
			want:      "0123456789",
		},
		{
			name:      "even ellipsis",
			input:     "0123456789",
			ellipsis:  "..",
			maxlength: 6,
			want:      "..6789",
		},
		{
			name:      "odd ellipsis",
			input:     "0123456789",
			ellipsis:  "...",
			maxlength: 7,
			want:      "...6789",
		},
		{
			name:      "too long ellipsis does not increase final length",
			input:     "0123456789",
			ellipsis:  "..........",
			maxlength: 6,
			want:      "456789",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateStartEllipsis(tt.input, tt.ellipsis, tt.maxlength)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
