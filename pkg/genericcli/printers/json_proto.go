package printers

import (
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoJSONPrinter prints data of type proto.Message in JSON format
type ProtoJSONPrinter struct {
	out      io.Writer
	fallback bool
}

func NewProtoJSONPrinter() *ProtoJSONPrinter {
	return &ProtoJSONPrinter{
		out: os.Stdout,
	}
}

func (p *ProtoJSONPrinter) WithOut(out io.Writer) *ProtoJSONPrinter {
	p.out = out
	return p
}

func (p *ProtoJSONPrinter) WithFallback(fallback bool) *ProtoJSONPrinter {
	p.fallback = fallback
	return p
}

func (p *ProtoJSONPrinter) Print(data any) error {
	msg, ok := data.(proto.Message)
	if !ok {
		if p.fallback {
			return NewJSONPrinter().WithOut(p.out).Print(data)
		}
		return fmt.Errorf("unable to marshal proto message because given data is not of type proto.Message")
	}

	m := &protojson.MarshalOptions{
		Indent: "    ",
	}
	content, err := m.Marshal(msg)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s\n", string(content))

	return nil
}
