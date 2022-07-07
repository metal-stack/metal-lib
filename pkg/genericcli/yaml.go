package genericcli

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

// ReadOne reads exactly one document from a multi-document YAML from a given path, returns an error if there are no or more than one documents in it
func (m *MultiDocumentYAML[D]) ReadOne(from string) (D, error) {
	emptyD := new(D)

	docs, err := m.ReadAll(from)
	if err != nil {
		return *emptyD, err
	}

	if len(docs) == 0 {
		return *emptyD, fmt.Errorf("no document parsed from yaml")
	}
	if len(docs) > 1 {
		return *emptyD, fmt.Errorf("more than one document parsed from yaml")
	}

	return docs[0], nil
}

// ReadIndex reads a document from a specific index of a multi-document YAML from a given path
func (m *MultiDocumentYAML[D]) ReadIndex(from string, index int) (D, error) {
	emptyD := new(D)

	err := validateFrom(m.fs, from)
	if err != nil {
		return *emptyD, err
	}

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
