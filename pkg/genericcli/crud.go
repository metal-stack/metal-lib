package genericcli

import (
	"fmt"
)

func (a *GenericCLI[C, U, R]) CreateFromFile(gneric Generic[C, U, R], from string) (R, error) {
	emptyR := new(R)

	mc := MultiDocumentYAML[C]{
		fs: a.fs,
	}

	doc, err := mc.ReadOne(from)
	if err != nil {
		return *emptyR, err
	}

	result, err := gneric.Create(doc)
	if err != nil {
		return *emptyR, fmt.Errorf("error creating entity: %w", err)
	}

	return *result, nil
}

func (a *GenericCLI[C, U, R]) UpdateFromFile(generic Generic[C, U, R], from string) (R, error) {
	emptyR := new(R)

	mc := MultiDocumentYAML[U]{
		fs: a.fs,
	}

	doc, err := mc.ReadOne(from)
	if err != nil {
		return *emptyR, err
	}

	result, err := generic.Update(doc)
	if err != nil {
		return *emptyR, fmt.Errorf("error updating entity: %w", err)
	}

	return result, nil
}
