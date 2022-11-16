package printers_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

type jsonPrinterTestExample struct {
	Str    string
	Num    int
	Real   float64
	Bool   bool
	Keys   []string
	Object map[string]string
}

func TestJsonSuccess(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewJSONPrinter().WithOut(buffer)
	err := printer.Print(jsonPrinterTestExample{
		"test", 42, 3.14, true, []string{"a", "b"}, map[string]string{
			"a": "b",
		},
	})
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `{
    "Str": "test",
    "Num": 42,
    "Real": 3.14,
    "Bool": true,
    "Keys": [
        "a",
        "b"
    ],
    "Object": {
        "a": "b"
    }
}
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestJsonPrintError(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewJSONPrinter().WithOut(buffer)
	err := printer.Print(fmt.Errorf("Test"))
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `"Test"
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}
