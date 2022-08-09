package genericcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/olekukonko/tablewriter"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	yaml "gopkg.in/yaml.v3"
)

type (
	Printer interface {
		Print(data any) error
	}

	// JSONPrinter prints data in JSON format
	JSONPrinter struct {
		out io.Writer
	}

	// YAMLPrinter prints data in YAML format
	YAMLPrinter struct {
		out io.Writer
	}

	// ProtoJSONPrinter prints data of type proto.Message in JSON format
	ProtoJSONPrinter struct {
		out io.Writer
	}

	// ProtoYAMLPrinter prints data of type proto.Message in YAML format
	ProtoYAMLPrinter struct {
		out io.Writer
	}

	// TemplatePrinter prints data with a given template
	TemplatePrinter struct {
		out io.Writer
		t   *template.Template
	}

	// TablePrinter prints data into a table
	TablePrinter struct {
		table *tablewriter.Table
		c     *TablePrinterConfig
	}

	// TablePrinterConfig contains the configuration for the table printer
	TablePrinterConfig struct {
		// ToHeaderAndRows is called during print to obtain the headers and rows for the given data.
		ToHeaderAndRows func(data any, wide bool) ([]string, [][]string, error)
		// Wide is passed to the headers and rows function and allows to provide extendend columns.
		Wide bool
		// Markdown will print the table in Markdown format
		Markdown bool
		// NoHeaders will omit headers during pring when set to true
		NoHeaders bool
		// Out defines the output writer for the printer, will default to os.stdout
		Out io.Writer
	}
)

func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{
		out: os.Stdout,
	}
}

func (p *JSONPrinter) Print(data any) error {
	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s\n", string(content))

	return nil
}

func NewProtoJSONPrinter() *ProtoJSONPrinter {
	return &ProtoJSONPrinter{
		out: os.Stdout,
	}
}

func (p *ProtoJSONPrinter) Print(data any) error {
	msg, ok := data.(proto.Message)
	if !ok {
		return fmt.Errorf("unable to marshal proto message because given data is not of type proto.Message")
	}

	m := &protojson.MarshalOptions{
		Indent: "    ",
	}
	content, err := m.Marshal(msg)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s\n", string(content))

	return nil
}

func NewYAMLPrinter() *YAMLPrinter {
	return &YAMLPrinter{
		out: os.Stdout,
	}
}

func (p *YAMLPrinter) Print(data any) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s", string(content))

	return nil
}

func NewProtoYAMLPrinter() *ProtoYAMLPrinter {
	return &ProtoYAMLPrinter{
		out: os.Stdout,
	}
}

func (p *ProtoYAMLPrinter) Print(data any) error {
	msg, ok := data.(proto.Message)
	if !ok {
		return fmt.Errorf("unable to marshal proto message because given data is not of type proto.Message")
	}

	intermediate, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	var r interface{}
	err = json.Unmarshal(intermediate, &r)
	if err != nil {
		return err
	}

	content, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s", string(content))

	return nil
}

func NewTemplatePrinter(t string) (*TemplatePrinter, error) {
	template, err := template.New("t").Funcs(sprig.TxtFuncMap()).Parse(t)
	if err != nil {
		return nil, err
	}

	return &TemplatePrinter{
		out: os.Stdout,
		t:   template,
	}, nil
}

func (p *TemplatePrinter) Print(data any) error {
	// first we transform the input to a struct which has fields with the same name as in the json struct.
	// this is handy for template rendering as the output of -o json|yaml can be used as the input for the template.
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if isSlice(data) {
		var d []any
		err = json.Unmarshal(raw, &d)
		if err != nil {
			return err
		}

		for _, elem := range d {
			err = p.print(elem)
			if err != nil {
				return err
			}
		}

		return nil
	}

	var d any
	err = json.Unmarshal(raw, &d)
	if err != nil {
		return err
	}

	err = p.print(d)
	if err != nil {
		return err
	}

	return nil
}

func (p *TemplatePrinter) print(data any) error {
	var buf bytes.Buffer

	err := p.t.Execute(&buf, data)
	if err != nil {
		return fmt.Errorf("unable to render template: %w", err)
	}

	fmt.Fprintf(p.out, "%s\n", buf.String())

	return nil
}

func isSlice(data any) bool {
	return reflect.ValueOf(data).Kind() == reflect.Slice
}

func NewTablePrinter(config *TablePrinterConfig) (*TablePrinter, error) {
	if config.ToHeaderAndRows == nil {
		return nil, fmt.Errorf("function for ")
	}
	if config.Out == nil {
		config.Out = os.Stdout
	}

	table := tablewriter.NewWriter(config.Out)

	if config.Markdown {
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
	} else {
		table.SetHeaderLine(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorder(false)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		table.SetRowLine(false)
		table.SetTablePadding("  ")
		table.SetNoWhiteSpace(true) // no whitespace in front of every line
	}

	return &TablePrinter{
		table: table,
		c:     config,
	}, nil
}

// GetTable can be used to alter the table element. Try not to do it all the time but rather propose an API change in this project.
func (p *TablePrinter) GetTable() *tablewriter.Table {
	return p.table
}

func (p *TablePrinter) Print(data any) error {
	header, rows, err := p.c.ToHeaderAndRows(data, p.c.Wide)
	if err != nil {
		return err
	}

	if !p.c.NoHeaders {
		p.table.SetHeader(header)
	}
	p.table.AppendBulk(rows)

	p.table.Render()

	return nil
}
