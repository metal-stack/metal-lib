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

		p printers.Printer
	}

	MultiApplyResults[R any] []MultiApplyResult[R]
)

func (m MultiApplyResult[R]) Append(ms MultiApplyResults[R], action MultiApplyAction, result *R, err error) MultiApplyResults[R] {
	if result != nil {
		m.Result = *result
	}
	m.Action = action
	m.Error = err

	m.Print()

	return append(ms, m)
}

func (m *MultiApplyResult[R]) Print() {
	if m.p == nil {
		return
	}

	if m.Error == nil {
		err := m.p.Print(m.Result)
		if err != nil {
			m.Error = err
		}
		return
	}

	err := m.p.Print(m.Error)
	if err != nil {
		m.Error = fmt.Errorf("error printing original error: %s, original error: %w", err, m.Error)
	}
}

func (ms MultiApplyResults[R]) ToList() []R {
	var result []R

	for _, m := range ms {
		if m.Error == nil {
			result = append(result, m.Result)
		}
	}

	return result
}

func (ms MultiApplyResults[R]) Error(joinErrors bool) error {
	var errors []string

	for _, m := range ms {
		if m.Error != nil {
			errors = append(errors, m.Error.Error())
		}
	}

	if len(errors) == 0 {
		return nil
	}

	if joinErrors {
		return fmt.Errorf("errors occurred during apply: %s", strings.Join(errors, ", "))
	}

	return fmt.Errorf("errors occurred during apply")
}

func AlreadyExistsError() error {
	return alreadyExistsError
}

// CreateFromFile creates a single entity from a given file containing a response entity.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) CreateFromFile(from string) (R, error) {
	var zero R

	doc, err := a.parser.ReadOne(from)
	if err != nil {
		return zero, err
	}

	rq, err := a.crud.ToCreate(doc)
	if err != nil {
		return zero, err
	}

	result, err := a.crud.Create(rq)
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

// UpdateFromFile updates a single entity from a given file containing a response entity.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) UpdateFromFile(from string) (R, error) {
	var zero R

	doc, err := a.parser.ReadOne(from)
	if err != nil {
		return zero, err
	}

	updateDoc, err := a.crud.ToUpdate(doc)
	if err != nil {
		return zero, err
	}

	result, err := a.crud.Update(updateDoc)
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

// ApplyFromFile creates or updates entities from a given file of response entities.
// In order to work, the create function must return an already exists error as defined in this package.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
//
// The printer can be passed optionally. If passed, results will be printed intermediately.
func (a *GenericCLI[C, U, R]) ApplyFromFile(from string, p printers.Printer) (MultiApplyResults[R], error) {
	docs, err := a.parser.ReadAll(from)
	if err != nil {
		return nil, err
	}

	var results MultiApplyResults[R]

	for index := range docs {
		var (
			res = MultiApplyResult[R]{p: p}
			doc = docs[index]
		)

		createDoc, err := a.crud.ToCreate(doc)
		if err != nil {
			results = res.Append(results, MultiApplyErrorOnCreate, nil, fmt.Errorf("error converting to create entity: %w", err))
			continue
		}

		created, err := a.crud.Create(createDoc)
		if err == nil {
			results = res.Append(results, MultiApplyCreated, &created, nil)
			continue
		}

		if !errors.Is(err, AlreadyExistsError()) {
			results = res.Append(results, MultiApplyErrorOnCreate, nil, fmt.Errorf("error creating entity: %w", err))
			continue
		}

		updateDoc, err := a.crud.ToUpdate(doc)
		if err != nil {
			results = res.Append(results, MultiApplyErrorOnUpdate, nil, fmt.Errorf("error converting to update entity: %w", err))
			continue
		}

		updated, err := a.crud.Update(updateDoc)
		if err == nil {
			results = res.Append(results, MultiApplyUpdated, &updated, nil)
			continue
		}

		results = res.Append(results, MultiApplyErrorOnUpdate, nil, fmt.Errorf("error updating entity: %w", err))
	}

	joinErrors := false
	if p == nil {
		joinErrors = true
	}

	return results, results.Error(joinErrors)
}

func (a *GenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p printers.Printer) error {
	if a.intermediateApplyPrint {
		_, err := a.ApplyFromFile(from, p)
		if err != nil {
			return err
		}

		return nil
	}

	var printErr error

	result, err := a.ApplyFromFile(from, nil)
	defer func() {
		printErr = p.Print(result.ToList())
	}()
	if err != nil {
		return err
	}

	return printErr
}
