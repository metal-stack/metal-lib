package e2e_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

	"github.com/spf13/cobra"

	"github.com/metal-stack/metal-lib/pkg/genericcli/e2e"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

type testData struct {
	Field1 string
	Field2 int
}

func TestFramework(t *testing.T) {
	tests := []*e2e.Test[testData, *testData]{
		{
			Name:       "run",
			CmdArgs:    []string{"test"},
			NewRootCmd: newRootCmd(nil),
			WantObject: &testData{
				Field1: "foo",
				Field2: 42,
			},
			WantProtoObject:          nil, // this is hard to test here
			AssertExhaustiveArgs:     true,
			AssertExhaustiveExcludes: []string{"output-format", "template"},
			WantTable: new(`
            FIELD 1
            foo
			`),
			WantWideTable: new(`
            FIELD 1  FIELD 2
            foo      42
			`),
			WantMarkdown: new(`
            | FIELD 1 |
            |---------|
            | foo     |
			`),
			WantTemplate: new(`foo`),
			Template:     new(`{{ .Field1 }}`),
			WantErr:      nil,
		},
		{
			Name:       "run",
			CmdArgs:    []string{"test"},
			NewRootCmd: newRootCmd(fmt.Errorf("a runtime error")),
			WantErr:    fmt.Errorf("a runtime error"),
		},
	}
	for _, tt := range tests {
		tt.TestCmd(t)
	}
}

func newRootCmd(err error) e2e.NewRootCmdFunc {
	return func() (*cobra.Command, *bytes.Buffer) {
		var (
			out     bytes.Buffer
			testCmd = &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					if err != nil {
						return err
					}

					var printer printers.Printer

					switch format := cmd.Flag("output-format").Value.String(); format {
					case "yaml":
						printer = printers.NewProtoYAMLPrinter().WithFallback(true).WithOut(&out)
					case "json":
						printer = printers.NewProtoJSONPrinter().WithFallback(true).WithOut(&out)
					case "yamlraw":
						printer = printers.NewYAMLPrinter().WithOut(&out)
					case "jsonraw":
						printer = printers.NewJSONPrinter().WithOut(&out)
					case "table", "wide", "markdown":
						printer = printers.NewTablePrinter(&printers.TablePrinterConfig{
							ToHeaderAndRows: func(data any, wide bool) ([]string, [][]string, error) {
								switch d := data.(type) {
								case *testData:
									if wide {
										return []string{"Field 1", "Field 2"}, [][]string{{d.Field1, strconv.Itoa(d.Field2)}}, nil
									}
									return []string{"Field 1"}, [][]string{{d.Field1}}, nil
								}
								return nil, nil, fmt.Errorf("no table printer for this type")
							},
							Wide:     format == "wide",
							Markdown: format == "markdown",
						}).WithOut(&out)
					case "template":
						printer = printers.NewTemplatePrinter(cmd.Flag("template").Value.String()).WithOut(&out)
					default:
						return fmt.Errorf("unknown output format: %q", format)
					}

					return printer.Print(&testData{
						Field1: "foo",
						Field2: 42,
					})
				},
			}
		)

		testCmd.PersistentFlags().StringP("output-format", "o", "table", "output format (table|wide|markdown|json|yaml|template|jsonraw|yamlraw).")
		testCmd.PersistentFlags().String("template", "", "for template output")

		return testCmd, &out
	}
}
