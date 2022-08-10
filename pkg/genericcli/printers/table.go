package printers

import (
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
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
	// NoHeaders will omit headers during pring when set to true
	NoHeaders bool
	// Out defines the output writer for the printer, will default to os.stdout
	Out io.Writer
}

func NewTablePrinter(config *TablePrinterConfig) *TablePrinter {
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
	}
}

func (p *TablePrinter) WithOut(out io.Writer) *tablewriter.Table {
	p.c.Out = out
	return p.table
}

// MutateTable can be used to alter the table element. Try not to do it all the time but rather propose an API change in this project.
func (p *TablePrinter) MutateTable(mutateFn func(table *tablewriter.Table)) {
	mutateFn(p.table)
}

func (p *TablePrinter) Print(data any) error {
	if p.c.ToHeaderAndRows == nil {
		return fmt.Errorf("missing to header and rows function in printer configuration")
	}

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
