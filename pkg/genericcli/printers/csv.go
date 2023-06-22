package printers

import (
	"encoding/csv"
	"io"
	"os"

	"github.com/jszwec/csvutil"
)

const defaultDelimiter = ';'

type CSVPrinter struct {
	c   *CSVPrinterConfig
	out io.Writer
}

type CSVPrinterConfig struct {
	// AutoHeader will generate headers during print, default is go standard ("false")
	AutoHeader bool
	// Delimiter the char to separate the columns, default is ";"
	Delimiter rune
	// Out defines the output writer for the printer, will default to os.stdout
	Out io.Writer
	// Tag sets the struct field tag used for printing (default: json)
	Tag string
}

func NewCSVPrinter(config *CSVPrinterConfig) *CSVPrinter {
	if config.Out == nil {
		config.Out = os.Stdout
	}

	if config.Tag == "" {
		config.Tag = "json"
	}

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
	w := csv.NewWriter(cp.out)
	w.Comma = cp.c.Delimiter

	enc := csvutil.NewEncoder(w)
	enc.AutoHeader = cp.c.AutoHeader
	enc.Tag = cp.c.Tag

	err := enc.Encode(data)
	if err != nil {
		return err
	}

	w.Flush()

	return nil
}
