package printers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// JSONPrinter prints data in JSON format
type JSONPrinter struct {
	out io.Writer
}

func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{
		out: os.Stdout,
	}
}

func (p *JSONPrinter) WithOut(out io.Writer) *JSONPrinter {
	p.out = out
	return p
}

func (p *JSONPrinter) Print(data any) error {
	if err, ok := data.(error); ok {
		data = err.Error()
	}

	content, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s\n", string(content))

	return nil
}
