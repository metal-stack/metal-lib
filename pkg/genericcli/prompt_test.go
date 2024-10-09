package genericcli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestPromptCustom(t *testing.T) {
	tests := []struct {
		name    string
		c       *PromptConfig
		input   string
		want    string
		wantErr error
	}{
		{
			name:  "default prompt config answered with yes",
			input: "yes\n",
			want:  "Do you want to continue? [y/n] ",
		},
		{
			name:    "default prompt config answered with no",
			input:   "no\n",
			want:    "Do you want to continue? [y/n] ",
			wantErr: fmt.Errorf(`aborting due to given answer ("no")`),
		},
		{
			name:  "custom prompt config",
			input: "ack\n",
			c: &PromptConfig{
				Message:         "Do you get it?",
				ShowAnswers:     true,
				AcceptedAnswers: []string{"ack", "a"},
				DefaultAnswer:   "ack",
				No:              "nack",
			},
			want: "Do you get it? [Ack/nack] ",
		},
		{
			name:  "custom prompt config, default answer with empty input",
			input: "\n",
			c: &PromptConfig{
				Message:         "Do you get it?",
				ShowAnswers:     true,
				AcceptedAnswers: []string{"ack", "a"},
				DefaultAnswer:   "ack",
				No:              "nack",
			},
			want: "Do you get it? [Ack/nack] ",
		},
		{
			name:  "custom prompt config, default is no answer",
			input: "ack\n",
			c: &PromptConfig{
				Message:         "Do you get it?",
				ShowAnswers:     true,
				AcceptedAnswers: []string{"ack", "a"},
				DefaultAnswer:   "nack",
				No:              "nack",
			},
			want: "Do you get it? [ack/Nack] ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				in  bytes.Buffer
				out bytes.Buffer
			)

			if tt.c == nil {
				tt.c = promptDefaultConfig()
			}
			tt.c.In = &in
			tt.c.Out = &out

			in.WriteString(tt.input)

			err := PromptCustom(tt.c)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}
			if diff := cmp.Diff(tt.want, out.String()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
