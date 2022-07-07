package cli

import (
	"fmt"

	"github.com/spf13/afero"
)

// Applier can be used to apply entities
type Applier[C any, U any, R any] struct {
	from string
	fs   afero.Fs
}

// Appliable must be implemented in order to apply entities
type Appliable[C any, U any, R any] interface {
	// Create tries to create the entity with the given request, if it already exists it does NOT return an error but nil for both return arguments.
	// if the creation was successful it returns the success response.
	Create(rq C) (*R, error)
	// Update tries to update the entity with the given request.
	// if the update was successful it returns the success response.
	Update(rq U) (R, error)
}

func NewApplier[C any, U any, R any](from string) (*Applier[C, U, R], error) {
	fs := afero.NewOsFs()

	switch from {
	case "":
		return nil, fmt.Errorf("from must not be empty")
	case "-":
	default:
		exists, err := afero.Exists(fs, from)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("file does not exist: %s", from)
		}
	}

	return &Applier[C, U, R]{
		from: from,
		fs:   fs,
	}, nil
}

func (a *Applier[C, U, R]) Apply(appliable Appliable[C, U, R]) ([]R, error) {
	mc := MultiDocumentYAML[C]{
		fs: a.fs,
	}

	docs, err := mc.ReadAll(a.from)
	if err != nil {
		return nil, err
	}

	result := []R{}
	mu := MultiDocumentYAML[U]{
		fs: a.fs,
	}

	for index := range docs {
		createDoc := docs[index]

		created, err := appliable.Create(createDoc)
		if err != nil {
			return nil, fmt.Errorf("error creating entity: %w", err)
		}

		if created != nil {
			result = append(result, *created)
			continue
		}

		updateDoc, err := mu.ReadIndex(a.from, index)
		if err != nil {
			return nil, err
		}

		updated, err := appliable.Update(updateDoc)
		if err != nil {
			return nil, fmt.Errorf("error updating entity: %w", err)
		}

		result = append(result, updated)
	}

	return result, nil
}
