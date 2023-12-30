package printers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"text/template"

	"github.com/go-task/slim-sprig/v3"
)

// TemplatePrinter prints data with a given template
type TemplatePrinter struct {
	out  io.Writer
	text string
	t    *template.Template
}

func NewTemplatePrinter(template string) *TemplatePrinter {
	return &TemplatePrinter{
		out:  os.Stdout,
		text: template,
	}
}

func (p *TemplatePrinter) WithOut(out io.Writer) *TemplatePrinter {
	p.out = out
	return p
}

func (p *TemplatePrinter) WithTemplate(t *template.Template) *TemplatePrinter {
	p.t = t
	return p
}

func (p *TemplatePrinter) Print(data any) error {
	if p.t == nil {
		var err error
		p.t, err = template.New("t").Funcs(sprig.TxtFuncMap()).Parse(p.text)
		if err != nil {
			return err
		}
	}

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
