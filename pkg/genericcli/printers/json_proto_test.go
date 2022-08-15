package printers_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers/proto_test"
)

func TestJsonProtoWithProto(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoJSONPrinter().WithOut(buffer)
	err := printer.Print(&proto_test.Foo{Text: "test"})
	if err != nil {
		t.Error(err)
	}
	// the proto response differs in whitespace from time to time
	got := strings.ReplaceAll(buffer.String(), " ", "")
	want := "{\n\"text\":\"test\"\n}\n"
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestJsonProtoWithJsonWithoutFallbackFails(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoJSONPrinter().
		WithOut(buffer).
		WithFallback(false)
	err := printer.Print(jsonPrinterTestExample{
		"test", 42, 3.14, true, []string{"a", "b"}, map[string]string{
			"a": "b",
		},
	})
	if err == nil {
		t.Error("want error because proto message is not of type proto.Message")
	}
	got := buffer.String()
	want := ""
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestJsonProtoWithJsonAndFallbackSucceeds(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoJSONPrinter().
		WithOut(buffer).
		WithFallback(true)
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
