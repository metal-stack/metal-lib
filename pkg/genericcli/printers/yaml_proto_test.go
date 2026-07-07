package printers_test

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers/proto_test"
)

func TestYamlProtoWithProto(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoYAMLPrinter().WithOut(buffer)
	err := printer.Print(&proto_test.Foo{
		Text:  "test",
		State: proto_test.State_STATE_ACTIVE,
		ListFoos: []*proto_test.NestedFoo{
			{
				Text: []string{"nested"},
			},
		},
		MapFoos: map[string]*proto_test.NestedFoo{
			"1": {Text: []string{"mapped"}},
		},
	})
	if err != nil {
		t.Error(err)
	}

	got := buffer.String()
	want := `text: test
state: STATE_ACTIVE
listFoos:
    - text:
        - nested
mapFoos:
    "1":
        text:
            - mapped
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
		t.Log("Use this for compare: \n" + buffer.String())
	}
}

func TestYamlProtoWithProtos(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoYAMLPrinter().WithFallback(false).WithOut(buffer)
	err := printer.Print([]*proto_test.Foo{
		{
			Text:  "test",
			State: proto_test.State_STATE_ACTIVE,
			ListFoos: []*proto_test.NestedFoo{
				{
					Text: []string{"nested"},
				},
			},
			MapFoos: map[string]*proto_test.NestedFoo{
				"1": {Text: []string{"mapped"}},
			},
		},
		{
			Text: "test2",
		},
	})
	if err != nil {
		t.Error(err)
	}

	got := buffer.String()
	want := `text: test
state: STATE_ACTIVE
listFoos:
    - text:
        - nested
mapFoos:
    "1":
        text:
            - mapped
---
text: test2
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
		t.Log("Use this for compare: \n" + buffer.String())
	}
}

func TestYamlProtoWithJsonWithoutFallbackFails(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoYAMLPrinter().
		WithOut(buffer).
		WithFallback(false)
	err := printer.Print(yamlPrinterTestExample{
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
		t.Log("Use this for compare: \n" + buffer.String())
	}
}

func TestYamlProtoWithJsonAndFallbackSucceeds(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewProtoYAMLPrinter().
		WithOut(buffer).
		WithFallback(true)
	err := printer.Print([]yamlPrinterTestExample{
		{
			"test", 42, 3.14, true, []string{"a", "b"}, map[string]string{
				"a": "b",
			},
		},
		{
			"test", 43, 3.14, true, []string{"a", "b"}, map[string]string{
				"a": "b",
			},
		},
	})
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `---
- bool: true
  keys:
  - a
  - b
  num: 42
  object:
    a: b
  real: 3.14
  str: test
- bool: true
  keys:
  - a
  - b
  num: 43
  object:
    a: b
  real: 3.14
  str: test
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
		t.Log("Use this for compare: \n" + buffer.String())
	}
}
