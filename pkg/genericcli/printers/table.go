package printers

import (
	"fmt"
	"io"
	"os"

	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

// TablePrinter prints data into a table
type TablePrinter struct {
	table *tablewriter.Table
	c     *TablePrinterConfig
}

// TablePrinterConfig contains the configuration for the table printer
type TablePrinterConfig struct {
	// ToHeaderAndRows is called during print to obtain the headers and rows for the given data.
	ToHeaderAndRows func(data any, wide bool) ([]string, [][]string, error)
	// Wide is passed to the headers and rows function and allows to provide extendend columns.
	Wide bool
	// Markdown will print the table in Markdown format
	Markdown bool
	// NoHeaders will omit headers during print when set to true
	NoHeaders bool
	// Out defines the output writer for the printer, will default to os.stdout
	Out io.Writer
	// CustomPadding defines the table padding, defaults to three whitespaces
	CustomPadding *string
	// DisableDefaultErrorPrinter disables the default error printer when the given print data is of type error.
	DisableDefaultErrorPrinter bool
}

func NewTablePrinter(config *TablePrinterConfig) *TablePrinter {
	if config.Out == nil {
		config.Out = os.Stdout
	}
	if config.CustomPadding == nil {
		config.CustomPadding = pointer.Pointer(" ")
	}

	return &TablePrinter{
		c: config,
	}
}

func (p *TablePrinter) WithOut(out io.Writer) *TablePrinter {
	p.c.Out = out
	return p
}

// MutateTable can be used to alter the table element. Try not to do it all the time but rather propose an API change in this project.
func (p *TablePrinter) MutateTable(mutateFn func(table *tablewriter.Table)) {
	mutateFn(p.table)
}

func (p *TablePrinter) Print(data any) error {
	if err, ok := data.(error); ok && !p.c.DisableDefaultErrorPrinter {
		fmt.Fprintf(p.c.Out, "%s\n", err)
		return nil
	}

	if err := p.initTable(); err != nil {
		return err
	}

	header, rows, err := p.c.ToHeaderAndRows(data, p.c.Wide)
	if err != nil {
		return err
	}

	if !p.c.NoHeaders {
		p.table.Header(header)
	}

	if err := p.table.Bulk(rows); err != nil {
		return err
	}

	if err := p.table.Render(); err != nil {
		return err
	}

	return nil
}

func (p *TablePrinter) initTable() error {
	if p.c.ToHeaderAndRows == nil {
		return fmt.Errorf("missing to header and rows function in printer configuration")
	}
	if p.c.CustomPadding == nil {
		return fmt.Errorf("padding must be set")
	}

	if p.c.Markdown {

		symbols := tw.NewSymbolCustom("Markdown").
			WithRow("-").
			WithColumn("|").
			WithCenter("|").
			WithMidLeft("|").
			WithMidRight("|")

		p.table = tablewriter.NewTable(p.c.Out,
			tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
				Borders: tw.Border{Left: tw.On, Top: tw.Off, Right: tw.On, Bottom: tw.Off},
				Symbols: symbols,
				Settings: tw.Settings{
					Lines:      tw.Lines{},
					Separators: tw.Separators{},
				},
			})),
		)
	} else {
		symbols := tw.NewSymbolCustom("Default").
			WithCenter("").WithColumn("").WithRow("")

		p.table = tablewriter.NewTable(p.c.Out,
			tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
				Borders: tw.BorderNone,
				Symbols: symbols,
				Settings: tw.Settings{
					Lines:      tw.Lines{},
					Separators: tw.Separators{},
				},
			})),
			tablewriter.WithConfig(tablewriter.Config{
				Header: tw.CellConfig{
					Formatting: tw.CellFormatting{
						Alignment: tw.AlignLeft,
					},
				},
				Row: tw.CellConfig{
					Formatting: tw.CellFormatting{
						Alignment: tw.AlignLeft,
					},
				},
			}),
		)
	}

	return nil
}
