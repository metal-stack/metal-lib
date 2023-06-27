package printers_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

type yamlPrinterTestExample struct {
	Str    string            `json:"str"`
	Num    int               `json:"num"`
	Real   float64           `json:"real"`
	Bool   bool              `json:"bool"`
	Keys   []string          `json:"keys"`
	Object map[string]string `json:"object"`
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
	got := buffer.String()
	want := `---
bool: true
keys:
- a
- b
num: 42
object:
  a: b
real: 3.14
str: test
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestYamlPrintError(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewYAMLPrinter().WithOut(buffer)
	err := printer.Print(fmt.Errorf("Test"))
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := "Test\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}
