package printers_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

func TestBasicTablePrinter(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out: buffer,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
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
	want := ` A  B 
 1  2 
 3  4 
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestBasicMarkdownTablePrinter(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out:      buffer,
		Markdown: true,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
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
	want := `| A | B |
|---|---|
| 1 | 2 |
| 3 | 4 |
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestBasicMarkdownTablePrinterWithoutHeaders(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out:       buffer,
		Markdown:  true,
		NoHeaders: true,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
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
	want := `| 1 | 2 |
| 3 | 4 |
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestBasicMarkdownTablePrinterWithoutHeadersAndRows(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out:      buffer,
		Markdown: true,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
			if data != "test" {
				t.Errorf("want data test, got %s", data)
			}
			return []string{"a", "b"}, [][]string{}, nil
		},
	})
	err := printer.Print("test")
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `| A | B |
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestMarkdownTablePrinterWithCustomOut(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Markdown: true,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
			if data != "test" {
				t.Errorf("want data test, got %s", data)
			}
			return []string{"a", "b"}, [][]string{
				{"1", "2"},
				{"3", "4"},
			}, nil
		},
	})
	printer.WithOut(buffer)
	err := printer.Print("test")
	if err != nil {
		t.Error(err)
	}
	got := buffer.String()
	want := `| A | B |
|---|---|
| 1 | 2 |
| 3 | 4 |
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestMarkdownTablePrinterWithWideOutput(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out:      buffer,
		Markdown: true,
		Wide:     true,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
			if data != "test" {
				t.Errorf("want data test, got %s", data)
			}
			if !wide {
				t.Errorf("want wide output")
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
	want := `| A | B |
|---|---|
| 1 | 2 |
| 3 | 4 |
`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestTableFailsWithMissingTOHeaderAndRows(t *testing.T) {
	buffer := new(bytes.Buffer)
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out: buffer,
	})
	err := printer.Print("test")
	if err == nil {
		t.Error("want error, got nil")
	}
	got := buffer.String()
	want := ``
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}

func TestTableFailsWithFailingContents(t *testing.T) {
	buffer := new(bytes.Buffer)
	expectedError := errors.New("want error")
	printer := printers.NewTablePrinter(&printers.TablePrinterConfig{
		Out: buffer,
		ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
			return nil, nil, expectedError
		},
	})
	err := printer.Print("test")
	if !errors.Is(err, expectedError) {
		t.Errorf("want error %q, got %q", expectedError, err)
	}
	got := buffer.String()
	want := ``
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("diff (+got -want):\n %s", diff)
	}
}
