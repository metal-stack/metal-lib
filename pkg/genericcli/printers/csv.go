package printers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/jszwec/csvutil"
)

type CSVPrinter struct {
	c   *CSVPrinterConfig
	out io.Writer
}

type CSVPrinterConfig struct {
	// AutoHeader will generate headers during print, default is go standard ("false")
	AutoHeader bool
	// Delimiter the char to separate the columns, default is ";"
	Delimiter rune
}

func NewCSVPrinter(config *CSVPrinterConfig) *CSVPrinter {
	const defaultDelimiter = ';'

	if config.Delimiter == 0 {
		config.Delimiter = defaultDelimiter
	}

	return &CSVPrinter{
		c: config,
	}
}

func (cp *CSVPrinter) WithOut(out io.Writer) *CSVPrinter {
	cp.out = out

	return cp
}

func (cp *CSVPrinter) Print(data any) error {
	var buf bytes.Buffer

	w := csv.NewWriter(cp.out)
	w.Comma = cp.c.Delimiter

	enc := csvutil.NewEncoder(w)
	enc.AutoHeader = cp.c.AutoHeader

	err := enc.Encode(data)
	if err != nil {
		return err
	}

	w.Flush()

	if err := w.Error(); err != nil {
		return err
	}

	fmt.Fprintf(cp.out, "%s\n", buf.String())

	return nil
}
