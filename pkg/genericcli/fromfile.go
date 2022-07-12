package genericcli

import (
	"errors"
	"fmt"
)

var alreadyExistsError = errors.New("entity already exists")

func AlreadyExistsError() error {
	return alreadyExistsError
}

func (a *GenericCLI[C, U, R]) CreateFromFile(from string) (R, error) {
	var zero R

	doc, err := a.createParser.ReadOne(from)
	if err != nil {
		return zero, err
	}

	result, err := a.crud.Create(doc)
	if err != nil {
		return zero, fmt.Errorf("error creating entity: %w", err)
	}

	return result, nil
}

func (a *GenericCLI[C, U, R]) CreateFromFileAndPrint(from string, p Printer) error {
	result, err := a.CreateFromFile(from)
	if err != nil {
		return err
	}

	return p.Print(result)
}

func (a *GenericCLI[C, U, R]) UpdateFromFile(from string) (R, error) {
	var zero R

	doc, err := a.updateParser.ReadOne(from)
	if err != nil {
		return zero, err
	}

	result, err := a.crud.Update(doc)
	if err != nil {
		return zero, fmt.Errorf("error updating entity: %w", err)
	}

	return result, nil
}

func (a *GenericCLI[C, U, R]) UpdateFromFileAndPrint(from string, p Printer) error {
	result, err := a.UpdateFromFile(from)
	if err != nil {
		return err
	}

	return p.Print(result)
}

// ApplyFromFile creates or updates entities from a given file.
// In order to work, the create function must return an already exists error as defined in this package.
func (a *GenericCLI[C, U, R]) ApplyFromFile(from string) ([]R, error) {
	docs, err := a.createParser.ReadAll(from)
	if err != nil {
		return nil, err
	}

	result := []R{}

	for index := range docs {
		createDoc := docs[index]

		created, err := a.crud.Create(createDoc)
		if err == nil {
			result = append(result, created)
			continue
		}

		if !errors.Is(err, AlreadyExistsError()) {
			return nil, fmt.Errorf("error creating entity: %w", err)
		}

		updateDoc, err := a.updateParser.ReadIndex(from, index)
		if err != nil {
			return nil, err
		}

		updated, err := a.crud.Update(updateDoc)
		if err != nil {
			return nil, fmt.Errorf("error updating entity: %w", err)
		}

		result = append(result, updated)
	}

	return result, nil
}

func (a *GenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p Printer) error {
	result, err := a.ApplyFromFile(from)
	if err != nil {
		return err
	}

	return p.Print(result)
}
