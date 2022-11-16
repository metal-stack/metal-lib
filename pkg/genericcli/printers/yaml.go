package printers

import (
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"
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
	if err, ok := data.(error); ok {
		data = err.Error()
	}

	content, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "---\n%s", string(content))

	return nil
}
