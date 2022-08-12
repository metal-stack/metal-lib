package printers_test

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

type yamlPrinterTestExample struct {
	Str    string
	Num    int
	Real   float64
	Bool   bool
	Keys   []string
	Object map[string]string
}

func TestYamlSuccess(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewYAMLPrinter().WithOut(buffer)
	err := printer.Print(yamlPrinterTestExample{
		"test", 42, 3.14, true, []string{"a", "b"}, map[string]string{
			"a": "b",
		},
	})
	if err != nil {
		t.Error(err)
	}
	actual := buffer.String()
	expected := `str: test
num: 42
real: 3.14
bool: true
keys:
    - a
    - b
object:
    a: b
`
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected diff: %s", diff)
	}
}
