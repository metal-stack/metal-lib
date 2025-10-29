package genericcli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/spf13/afero"
)

func TestList(t *testing.T) {
	tests := []struct {
		name       string
		fileMockFn func(fs afero.Fs)
		want       []*Context
		wantOut    string
		wantErr    error
	}{
		{
			name: "list non-existent file",
			fileMockFn: func(fs afero.Fs) {

			},
			want:    nil,
			wantErr: errors.New("you need to create a context first"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.Buffer{}

			c := &ContextConfig{
				ConfigDirName:         fmt.Sprintf("./.%s", os.Args[0]),
				BinaryName:            os.Args[0],
				Fs:                    afero.NewMemMapFs(),
				In:                    nil,
				Out:                   &buf,
				ProjectListCompletion: nil,
			}

			tablePrinter := newPrinterFromCLI(c)

			c.ListPrinter = func() printers.Printer { return tablePrinter }
			c.DescribePrinter = func() printers.Printer { return tablePrinter }

			cmd := NewContextCmd(c)
			os.Args = []string{"metalctlv2", "list"}

			// TODO compare with _want_
			err := cmd.Execute()
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
				return
			}
			if diff := cmp.Diff(tt.wantOut, buf.String()); diff != "" {
				t.Errorf("Diff = %s", diff)
			}
		})
	}
}
