package printers

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"buf.build/go/protoyaml"
	"google.golang.org/protobuf/proto"
)

// ProtoYAMLPrinter prints data of type proto.Message in YAML format
type ProtoYAMLPrinter struct {
	out      io.Writer
	fallback bool
}

func NewProtoYAMLPrinter() *ProtoYAMLPrinter {
	return &ProtoYAMLPrinter{
		out: os.Stdout,
	}
}

func (p *ProtoYAMLPrinter) WithOut(out io.Writer) *ProtoYAMLPrinter {
	p.out = out
	return p
}

func (p *ProtoYAMLPrinter) WithFallback(fallback bool) *ProtoYAMLPrinter {
	p.fallback = fallback
	return p
}

func (p *ProtoYAMLPrinter) Print(data any) error {
	val := reflect.ValueOf(data)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		if val.Len() == 0 {
			return nil
		}

		items := make([]proto.Message, 0, val.Len())

		for i := range val.Len() {
			msg, ok := val.Index(i).Interface().(proto.Message)
			if !ok {
				if p.fallback {
					return NewYAMLPrinter().WithOut(p.out).Print(data)
				}

				return fmt.Errorf("unable to marshal proto message because element at index %d is not of type proto.Message", i)
			}

			items = append(items, msg)
		}

		for i, doc := range items {
			content, err := protoyaml.Marshal(doc)
			if err != nil {
				return err
			}

			if i > 0 {
				_, _ = fmt.Fprint(p.out, "---\n")
			}

			_, _ = p.out.Write(content)
		}

		return nil

	default:
		msg, ok := data.(proto.Message)
		if !ok {
			if p.fallback {
				return NewYAMLPrinter().WithOut(p.out).Print(data)
			}
			return fmt.Errorf("unable to marshal proto message because given data is not of type proto.Message")
		}

		content, err := protoyaml.Marshal(msg)
		if err != nil {
			return err
		}

		_, _ = fmt.Fprintf(p.out, "%s", string(content))

		return nil
	}
}
