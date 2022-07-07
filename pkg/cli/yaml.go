package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v3"
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
	reader, err := getReader(m.fs, from)
	if err != nil {
		return nil, err
	}

	var docs []D

	dec := yaml.NewDecoder(reader)

	for {
		data := new(D)

		err := dec.Decode(&data)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decode error: %w", err)
		}

		docs = append(docs, *data)
	}

	return docs, nil
}

// ReadIndex reads a document from a specific index of a multi-document YAML from a given path
func (m *MultiDocumentYAML[D]) ReadIndex(from string, index int) (D, error) {
	emptyD := new(D)

	reader, err := getReader(m.fs, from)
	if err != nil {
		return *emptyD, err
	}

	dec := yaml.NewDecoder(reader)

	count := 0
	for {
		data := new(D)

		err := dec.Decode(data)
		if errors.Is(err, io.EOF) {
			return *emptyD, fmt.Errorf("index not found in document: %d", index)
		}
		if err != nil {
			return *emptyD, fmt.Errorf("decode error: %w", err)
		}

		if count == index {
			return *data, nil
		}

		count++
	}
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
