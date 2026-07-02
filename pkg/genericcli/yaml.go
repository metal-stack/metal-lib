package genericcli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"buf.build/go/protoyaml"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"google.golang.org/protobuf/proto"

	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

// MultiDocumentYAML offers functions on multidocument YAML files
type MultiDocumentYAML[D any] struct {
	fs afero.Fs
}

func NewMultiDocumentYAML[D any]() *MultiDocumentYAML[D] {
	return &MultiDocumentYAML[D]{
		fs: afero.NewOsFs(),
	}
}

// ReadAll reads all documents from a multi-document YAML from a given path
func (m *MultiDocumentYAML[D]) ReadAll(from string) ([]D, error) {
	err := validateFrom(m.fs, from)
	if err != nil {
		return nil, err
	}

	ioReader, err := getReader(m.fs, from)
	if err != nil {
		return nil, err
	}

	var (
		docs    []D
		isProto = isProtoMsg[D]()

		bufReader  = bufio.NewReader(ioReader)
		yamlReader = utilyaml.NewYAMLReader(bufReader)
	)

	for {
		docBytes, err := yamlReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode error: %w", err)
		}

		if strings.TrimSpace(string(docBytes)) == "" {
			continue
		}

		var data D

		if isProto {
			var err error
			data, err = unmarshalProto[D](docBytes)
			if err != nil {
				return nil, fmt.Errorf("unable to unmarshal document from multi-yaml protoyaml document: %w", err)
			}
		} else if err := yaml.Unmarshal(docBytes, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal document from multi-yaml document: %w", err)
		}

		docs = append(docs, data)
	}

	return docs, nil
}

// ReadOne reads exactly one document from a multi-document YAML from a given path, returns an error if there are no or more than one documents in it
func (m *MultiDocumentYAML[D]) ReadOne(from string) (D, error) {
	var zero D

	docs, err := m.ReadAll(from)
	if err != nil {
		return zero, err
	}

	if len(docs) == 0 {
		return zero, fmt.Errorf("no document parsed from yaml")
	}
	if len(docs) > 1 {
		return zero, fmt.Errorf("more than one document parsed from yaml")
	}

	return docs[0], nil
}

// ReadIndex reads a document from a specific index of a multi-document YAML from a given path
func (m *MultiDocumentYAML[D]) ReadIndex(from string, index int) (D, error) {
	var zero D

	err := validateFrom(m.fs, from)
	if err != nil {
		return zero, err
	}

	ioReader, err := getReader(m.fs, from)
	if err != nil {
		return zero, err
	}

	var (
		count   = 0
		isProto = isProtoMsg[D]()

		bufReader  = bufio.NewReader(ioReader)
		yamlReader = utilyaml.NewYAMLReader(bufReader)
	)

	for {
		docBytes, err := yamlReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return zero, fmt.Errorf("index not found in document: %d", index)
			}
			return zero, fmt.Errorf("decode error: %w", err)
		}

		if strings.TrimSpace(string(docBytes)) == "" {
			continue
		}

		var data D

		if isProto {
			var err error
			data, err = unmarshalProto[D](docBytes)
			if err != nil {
				return zero, fmt.Errorf("unable to decode index %d from multi-yaml protoyaml document: %w", index, err)
			}
		} else if err := yaml.Unmarshal(docBytes, &data); err != nil {
			return zero, fmt.Errorf("unable to decode index %d from multi-yaml document: %w", index, err)
		}

		if count == index {
			return data, nil
		}

		count++
	}
}

// YamlIsEqual returns true if a yaml equal in content.
func YamlIsEqual(x []byte, y []byte) (bool, error) {
	var xParsed any
	err := yaml.Unmarshal(x, &xParsed)
	if err != nil {
		return false, err
	}

	var yParsed any
	err = yaml.Unmarshal(y, &yParsed)
	if err != nil {
		return false, err
	}

	return cmp.Equal(xParsed, yParsed), nil
}

func getReader(fs afero.Fs, from string) (io.Reader, error) {
	var reader io.Reader
	var err error
	switch from {
	case "-":
		reader = os.Stdin
	default:
		reader, err = fs.Open(from)
		if err != nil {
			return nil, fmt.Errorf("unable to open %q: %w", from, err)
		}
	}

	return reader, nil
}

func validateFrom(fs afero.Fs, from string) error {
	switch from {
	case "":
		return fmt.Errorf("from must not be empty")
	case "-":
	default:
		exists, err := afero.Exists(fs, from)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("file does not exist: %s", from)
		}
	}

	return nil
}

func isProtoMsg[D any]() bool {
	t := reflect.TypeFor[D]()
	v := protoValue(t)
	_, ok := v.Interface().(proto.Message)
	return ok
}

func unmarshalProto[D any](docBytes []byte) (D, error) {
	var zero D
	t := reflect.TypeFor[D]()
	pm := protoValue(t).Interface().(proto.Message)
	if err := protoyaml.Unmarshal(docBytes, pm); err != nil {
		return zero, err
	}
	return pm.(D), nil
}

func protoValue(t reflect.Type) reflect.Value {
	if t.Kind() == reflect.Pointer {
		return reflect.New(t.Elem())
	}
	return reflect.New(t)
}
