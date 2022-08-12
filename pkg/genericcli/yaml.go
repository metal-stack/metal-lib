package genericcli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	yaml "github.com/goccy/go-yaml" // we do not use the standard yaml library from go because it does not support json tags
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	defaultyaml "gopkg.in/yaml.v3"
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

	reader, err := getReader(m.fs, from)
	if err != nil {
		return nil, err
	}

	var docs []D

	dec := yaml.NewDecoder(reader)

	for {
		// go-yaml does not parse into a slice of pointer structs (result into nil)
		// therefore we parse yaml into a map and then put it into the final object with json
		var intermediate any
		err := dec.Decode(&intermediate)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode error: %w", err)
		}

		bytes, err := json.Marshal(intermediate)
		if err != nil {
			return nil, err
		}

		var data D
		err = json.Unmarshal(bytes, &data)
		if err != nil {
			return nil, fmt.Errorf("decode error: %w", err)
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

	reader, err := getReader(m.fs, from)
	if err != nil {
		return zero, err
	}

	dec := yaml.NewDecoder(reader)

	count := 0
	for {
		// go-yaml does not parse into a slice of pointer structs (result into nil)
		// therefore we parse yaml into a map and then put it into the final object with json
		var intermediate any
		err := dec.Decode(&intermediate)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return zero, fmt.Errorf("index not found in document: %d", index)
			}
			return zero, fmt.Errorf("decode error: %w", err)
		}

		bytes, err := json.Marshal(intermediate)
		if err != nil {
			return zero, err
		}

		var data D
		err = json.Unmarshal(bytes, &data)
		if err != nil {
			return zero, fmt.Errorf("decode error: %w", err)
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
	err := defaultyaml.Unmarshal(x, &xParsed)
	if err != nil {
		return false, err
	}

	var yParsed any
	err = defaultyaml.Unmarshal(y, &yParsed)
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
