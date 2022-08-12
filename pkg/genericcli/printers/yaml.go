package printers

import (
	"fmt"
	"io"
	"os"

	yaml "github.com/goccy/go-yaml" // we do not use the standard yaml library from go because it does not support json tags
)

// YAMLPrinter prints data in YAML format
type YAMLPrinter struct {
	out io.Writer
}

func NewYAMLPrinter() *YAMLPrinter {
	return &YAMLPrinter{
		out: os.Stdout,
	}
}

func (p *YAMLPrinter) WithOut(out io.Writer) *YAMLPrinter {
	p.out = out
	return p
}

func (p *YAMLPrinter) Print(data any) error {
	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s", string(content))

	return nil
}
