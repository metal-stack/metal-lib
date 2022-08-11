package genericcli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

var alreadyExistsError = errors.New("entity already exists")

const (
	MultiApplyCreated       MultiApplyAction = "created"
	MultiApplyUpdated       MultiApplyAction = "updated"
	MultiApplyErrorOnCreate MultiApplyAction = "error_on_create"
	MultiApplyErrorOnUpdate MultiApplyAction = "error_on_update"
)

type (
	MultiApplyAction string

	MultiApplyResult[R any] struct {
		Result R
		Action MultiApplyAction
		Error  error
	}

	MultiApplyResults[R any] []MultiApplyResult[R]
)

func (ms MultiApplyResults[R]) ToList() []R {
	var result []R

	for _, m := range ms {
		if m.Error == nil {
			result = append(result, m.Result)
		}
	}

	return result
}

func (ms MultiApplyResults[R]) Error() error {
	var errors []string

	for _, m := range ms {
		if m.Error != nil {
			errors = append(errors, m.Error.Error())
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf("errors occurred during apply: %s", strings.Join(errors, ", "))
}

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

func (a *GenericCLI[C, U, R]) CreateFromFileAndPrint(from string, p printers.Printer) error {
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

func (a *GenericCLI[C, U, R]) UpdateFromFileAndPrint(from string, p printers.Printer) error {
	result, err := a.UpdateFromFile(from)
	if err != nil {
		return err
	}

	return p.Print(result)
}

// ApplyFromFile creates or updates entities from a given file.
// In order to work, the create function must return an already exists error as defined in this package.
func (a *GenericCLI[C, U, R]) ApplyFromFile(from string) (MultiApplyResults[R], error) {
	docs, err := a.createParser.ReadAll(from)
	if err != nil {
		return nil, err
	}

	var result MultiApplyResults[R]

	for index := range docs {
		createDoc := docs[index]

		created, err := a.crud.Create(createDoc)
		if err == nil {
			result = append(result, MultiApplyResult[R]{Action: MultiApplyCreated, Result: created})
			continue
		}

		if !errors.Is(err, AlreadyExistsError()) {
			result = append(result, MultiApplyResult[R]{Action: MultiApplyErrorOnCreate, Error: fmt.Errorf("error creating entity: %w", err)})
			continue
		}

		updateDoc, err := a.updateParser.ReadIndex(from, index)
		if err != nil {
			return nil, err
		}

		updated, err := a.crud.Update(updateDoc)
		if err != nil {
			result = append(result, MultiApplyResult[R]{Action: MultiApplyErrorOnUpdate, Error: fmt.Errorf("error updating entity: %w", err)})
			continue
		}

		result = append(result, MultiApplyResult[R]{Action: MultiApplyUpdated, Result: updated})
	}

	return result, result.Error()
}

func (a *GenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p printers.Printer) error {
	var printErr error

	result, err := a.ApplyFromFile(from)
	defer func() {
		printErr = p.Print(result.ToList())
	}()
	if err != nil {
		return err
	}

	return printErr
}
