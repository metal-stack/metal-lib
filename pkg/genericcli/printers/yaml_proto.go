package printers

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	yaml "github.com/goccy/go-yaml" // we do not use the standard yaml library from go because it does not support json tags
	"google.golang.org/protobuf/encoding/protojson"
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
	msg, ok := data.(proto.Message)
	if !ok {
		if p.fallback {
			return NewYAMLPrinter().Print(data)
		}
		return fmt.Errorf("unable to marshal proto message because given data is not of type proto.Message")
	}

	intermediate, err := protojson.Marshal(msg)
	if err != nil {
		return err
	}

	var r any
	err = json.Unmarshal(intermediate, &r)
	if err != nil {
		return err
	}

	content, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	fmt.Fprintf(p.out, "%s", string(content))

	return nil
}
