package printers_test

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

func TestBasicCSVPrinter(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewCSVPrinter(&printers.CSVPrinterConfig{
		Out: buffer,
		ToHeaderAndRows: func(data any) ([]string, [][]string, error) {
			if data != "test" {
				t.Errorf("want data test, got %s", data)
			}
			return []string{"a", "b"}, [][]string{
				{"1", "2"},
				{"3", "4"},
			}, nil
		},
	})

	err := printer.Print("test")
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `a;b
1;2
3;4
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestBasicCSVPrinterWithoutHeader(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewCSVPrinter(&printers.CSVPrinterConfig{
		Out: buffer,
		ToHeaderAndRows: func(data any) ([]string, [][]string, error) {
			if data != "test" {
				t.Errorf("want data test, got %s", data)
			}
			return []string{"a", "b"}, [][]string{
				{"1", "2"},
				{"3", "4"},
			}, nil
		},
		NoHeaders: true,
	})

	err := printer.Print("test")
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `1;2
3;4
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestBasicCSVPrinterWithCustomDelimiter(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewCSVPrinter(&printers.CSVPrinterConfig{
		Out: buffer,
		ToHeaderAndRows: func(data any) ([]string, [][]string, error) {
			if data != "test" {
				t.Errorf("want data test, got %s", data)
			}
			return []string{"a", "b"}, [][]string{
				{"1", "2"},
				{"3", "4"},
			}, nil
		},
		Delimiter: ',',
	})

	err := printer.Print("test")
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `a,b
1,2
3,4
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}
