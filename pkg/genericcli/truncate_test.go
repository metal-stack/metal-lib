package genericcli

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTruncateMiddleElipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		elipsis   string
		maxlength int
		want      string
	}{
		{
			name:      "no trunc on short enough input",
			input:     "0123456789",
			elipsis:   TruncateElipsis,
			maxlength: 10,
			want:      "0123456789",
		},
		{
			name:      "even elipsis is in the middle on even length",
			input:     "0123456789",
			elipsis:   "..",
			maxlength: 6,
			want:      "01..89",
		},
		{
			name:      "even elipsis is slightly to the right on odd length",
			input:     "0123456789",
			elipsis:   "..",
			maxlength: 7,
			want:      "012..89",
		},
		{
			name:      "odd elipsis is in the middle on odd length",
			input:     "0123456789",
			elipsis:   TruncateElipsis,
			maxlength: 7,
			want:      "01...89",
		},
		{
			name:      "odd elipsis is slightly on the right on even length",
			input:     "0123456789",
			elipsis:   TruncateElipsis,
			maxlength: 6,
			want:      "01...9",
		},
		{
			name:      "too long elipsis does not increase final length",
			input:     "0123456789",
			elipsis:   "..........",
			maxlength: 6,
			want:      "012345",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateMiddleElipsis(tt.input, tt.elipsis, tt.maxlength)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestTruncateEndElipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		elipsis   string
		maxlength int
		want      string
	}{
		{
			name:      "no trunc on short enough input",
			input:     "0123456789",
			elipsis:   TruncateElipsis,
			maxlength: 10,
			want:      "0123456789",
		},
		{
			name:      "even elipsis",
			input:     "0123456789",
			elipsis:   "..",
			maxlength: 6,
			want:      "0123..",
		},
		{
			name:      "odd elipsis",
			input:     "0123456789",
			elipsis:   "...",
			maxlength: 7,
			want:      "0123...",
		},
		{
			name:      "too long elipsis does not increase final length",
			input:     "0123456789",
			elipsis:   "..........",
			maxlength: 6,
			want:      "012345",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateEndElipsis(tt.input, tt.elipsis, tt.maxlength)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestTruncateStartElipsis(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		elipsis   string
		maxlength int
		want      string
	}{
		{
			name:      "no trunc on short enough input",
			input:     "0123456789",
			elipsis:   TruncateElipsis,
			maxlength: 10,
			want:      "0123456789",
		},
		{
			name:      "even elipsis",
			input:     "0123456789",
			elipsis:   "..",
			maxlength: 6,
			want:      "..6789",
		},
		{
			name:      "odd elipsis",
			input:     "0123456789",
			elipsis:   "...",
			maxlength: 7,
			want:      "...6789",
		},
		{
			name:      "too long elipsis does not increase final length",
			input:     "0123456789",
			elipsis:   "..........",
			maxlength: 6,
			want:      "456789",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateStartElipsis(tt.input, tt.elipsis, tt.maxlength)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
