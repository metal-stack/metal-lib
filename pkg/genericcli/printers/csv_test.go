package printers_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

type CSVReport struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

var (
	content = []CSVReport{
		{
			FirstName: "John",
			LastName:  "Deere",
			Username:  "jd",
		},
		{
			FirstName: "New",
			LastName:  "Holland",
			Username:  "nh",
		},
	}
	expectationWithHeader = `FirstName;LastName;Username
John;Deere;jd
New;Holland;nh

`
	expectationWithOutHeader = `John;Deere;jd
New;Holland;nh

`
	delimiter = ';' // rune(59) ≙ ';'

)

func TestCSVWithHeader(t *testing.T) {
	t.Parallel()

	out := new(bytes.Buffer)
	config := &printers.CSVPrinterConfig{
		Delimiter:  delimiter,
		AutoHeader: true,
	}
	printer := printers.NewCSVPrinter(config).WithOut(out)

	err := printer.Print(content)
	if err != nil {
		t.Error(err)
	}

	got := out.String()
	want := expectationWithHeader

	if !reflect.DeepEqual(got, want) {
		diff := cmp.Diff(want, got)
		t.Errorf("got %v, want %v, diff %s", got, want, diff)
	}
}

func TestCSVWithHeaderNoDelimiter(t *testing.T) {
	t.Parallel()

	out := new(bytes.Buffer)
	config := &printers.CSVPrinterConfig{
		AutoHeader: true,
	}
	printer := printers.NewCSVPrinter(config).WithOut(out)

	err := printer.Print(content)
	if err != nil {
		t.Error(err)
	}

	got := out.String()
	want := expectationWithHeader

	if !reflect.DeepEqual(got, want) {
		diff := cmp.Diff(want, got)
		t.Errorf("got %v, want %v, diff %s", got, want, diff)
	}
}

func TestCSVNoHeader(t *testing.T) {
	t.Parallel()

	out := new(bytes.Buffer)

	config := &printers.CSVPrinterConfig{
		AutoHeader: false,
		Delimiter:  delimiter,
	}
	printer := printers.NewCSVPrinter(config).WithOut(out)

	err := printer.Print(content)
	if err != nil {
		t.Error(err)
	}

	got := out.String()
	want := expectationWithOutHeader

	if !reflect.DeepEqual(got, want) {
		diff := cmp.Diff(want, got)
		t.Errorf("got %v, want %v, diff %s", got, want, diff)
	}
}

func TestCSVEmptyArgHeader(t *testing.T) {
	t.Parallel()

	out := new(bytes.Buffer)

	config := &printers.CSVPrinterConfig{
		Delimiter: delimiter,
	}
	printer := printers.NewCSVPrinter(config).WithOut(out)

	err := printer.Print(content)
	if err != nil {
		t.Error(err)
	}

	got := out.String()
	want := expectationWithOutHeader

	if !reflect.DeepEqual(got, want) {
		diff := cmp.Diff(want, got)
		t.Errorf("got %v, want %v, diff %s", got, want, diff)
	}
}
