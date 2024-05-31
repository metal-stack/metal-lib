package printers

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func TestTemplatePrinter_Print(t *testing.T) {
	type nested struct {
		C string `json:"c"`
	}
	type machine struct {
		A      string `json:"a"`
		B      string `json:"b"`
		Nested nested `json:"nested"`
	}

	tests := []struct {
		name    string
		t       string
		data    any
		want    string
		wantErr error
	}{
		{
			name:    "template single entity",
			t:       "{{ .a }} {{ .b }} {{ .nested.c }}",
			data:    machine{A: "a", B: "b", Nested: nested{C: "c"}},
			want:    "a b c\n",
			wantErr: nil,
		},
		{
			name:    "template multiple entities",
			t:       "{{ .a }} {{ .b }}",
			data:    []machine{{A: "a", B: "b"}, {A: "c", B: "d"}},
			want:    "a b\nc d\n",
			wantErr: nil,
		},
		{
			name:    "also works with list of pointers",
			t:       "{{ .a }} {{ .b }}",
			data:    []*machine{{A: "a", B: "b"}, {A: "c", B: "d"}},
			want:    "a b\nc d\n",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			p := NewTemplatePrinter(tt.t).WithOut(&out)

			err := p.Print(tt.data)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(tt.want, out.String()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func TestTemplatePrinter_WithTemplate(t *testing.T) {
	type machine struct {
		A string `json:"a"`
		B string `json:"b"`
	}

	wrong := "{{ .a }}"
	correct, err := template.New("test").Parse("{{ .a }} {{ .b }}")
	if err != nil {
		t.Error(err)
	}
	var out bytes.Buffer
	p := NewTemplatePrinter(wrong).
		WithOut(&out).
		WithTemplate(correct)

	err = p.Print(machine{A: "a", B: "b"})
	if err != nil {
		t.Error(err)
	}
	want := "a b\n"
	got := out.String()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestTemplatePrinter_OmitEmptyRowsInSliceResponses(t *testing.T) {
	type obj struct {
		Name string `json:"name"`
	}

	var (
		out      bytes.Buffer
		out2     bytes.Buffer
		tpl      = `{{ if eq .name "test" }}{{ .name }}{{ end }}`
		testObjs = []obj{{Name: "a"}, {Name: "test"}}
	)

	p := NewTemplatePrinter(tpl).WithOut(&out)
	err := p.Print(testObjs)
	if err != nil {
		t.Error(err)
	}

	want := "test\n"
	got := out.String()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}

	p = NewTemplatePrinter(tpl).WithOut(&out2).WithoutOmitEmptyLines()
	err = p.Print(testObjs)
	if err != nil {
		t.Error(err)
	}

	want = "\ntest\n"
	got = out2.String()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}
