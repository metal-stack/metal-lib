package printers

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultDelimiter = ';'

type CSVPrinter struct {
	c *CSVPrinterConfig
}

type CSVPrinterConfig struct {
	// ToHeaderAndRows is called during print to obtain the headers and rows for the given data.
	ToHeaderAndRows func(data any) ([]string, [][]string, error)
	// NoHeaders will omit headers during pring when set to true
	NoHeaders bool
	// Out defines the output writer for the printer, will default to os.stdout
	Out io.Writer
	// Delimiter the char to separate the columns, default is ";"
	Delimiter rune
}

func NewCSVPrinter(config *CSVPrinterConfig) *CSVPrinter {
	if config == nil {
		config = &CSVPrinterConfig{}
	}

	if config.Out == nil {
		config.Out = os.Stdout
	}

	if config.Delimiter == 0 {
		config.Delimiter = defaultDelimiter
	}

	return &CSVPrinter{
		c: config,
	}
}

func (cp *CSVPrinter) WithOut(out io.Writer) *CSVPrinter {
	cp.c.Out = out

	return cp
}

func (cp *CSVPrinter) Print(data any) error {
	if cp.c.ToHeaderAndRows == nil {
		return fmt.Errorf("missing to header and rows function in printer configuration")
	}

	headers, rows, err := cp.c.ToHeaderAndRows(data)
	if err != nil {
		return err
	}

	if !cp.c.NoHeaders {
		fmt.Fprintln(cp.c.Out, strings.Join(headers, string(cp.c.Delimiter)))
	}

	for _, row := range rows {
		fmt.Fprintln(cp.c.Out, strings.Join(row, string(cp.c.Delimiter)))
	}

	return nil
}
