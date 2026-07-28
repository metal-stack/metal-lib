package printers_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/genericcli/teststructs"
)

func TestJsonProtoWithProto(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoJSONPrinter().WithOut(buffer)
	err := printer.Print(&teststructs.Foo{
		Text:  "test",
		State: teststructs.State_STATE_ACTIVE,
		ListFoos: []*teststructs.NestedFoo{
			{
				Text: []string{"nested"},
			},
		},
		MapFoos: map[string]*teststructs.NestedFoo{
			"1": {Text: []string{"mapped"}},
		},
	})
	if err != nil {
		t.Error(err)
	}

	want := `
{
    "text":  "test",
    "state":  "STATE_ACTIVE",
    "listFoos":  [
        {
            "text":  [
                "nested"
            ]
        }
    ],
    "mapFoos":  {
        "1":  {
            "text":  [
                "mapped"
            ]
        }
    }
}`

	// the proto response differs in whitespace from time to time
	got := strings.ReplaceAll(strings.TrimSpace(buffer.String()), " ", "")
	want = strings.ReplaceAll(strings.TrimSpace(want), " ", "")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
		t.Log("Use this for compare: \n" + buffer.String())
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
