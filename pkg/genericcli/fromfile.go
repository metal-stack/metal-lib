package genericcli

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
)

var alreadyExistsError = errors.New("entity already exists")

func AlreadyExistsError() error {
	return alreadyExistsError
}

const (
	MultiApplyCreated       MultiApplyAction = "created"
	MultiApplyUpdated       MultiApplyAction = "updated"
	MultiApplyDeleted       MultiApplyAction = "deleted"
	MultiApplyErrorOnCreate MultiApplyAction = "error_on_create"
	MultiApplyErrorOnUpdate MultiApplyAction = "error_on_update"
	MultiApplyErrorOnDelete MultiApplyAction = "error_on_delete"
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

func (m *MultiApplyResult[R]) Print(p printers.Printer) {
	if p == nil {
		return
	}

	if m.Error == nil {
		err := p.Print(m.Result)
		if err != nil {
			m.Error = err
		}
		return
	}

	err := p.Print(m.Error)
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
		return fmt.Errorf("%s", strings.Join(errors, ", "))
	}

	return fmt.Errorf("errors occurred during the process")
}

// CreateFromFile creates entities from a given file containing response entities.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) CreateFromFile(from string) (MultiApplyResults[R], error) {
	return a.multiOperation(from, multiOperationCreate, true)
}

func (a *GenericCLI[C, U, R]) CreateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, multiOperationCreate, p)
}

// UpdateFromFile updates entities from a given file containing response entities.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) UpdateFromFile(from string) (MultiApplyResults[R], error) {
	return a.multiOperation(from, multiOperationUpdate, true)

}

func (a *GenericCLI[C, U, R]) UpdateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, multiOperationUpdate, p)
}

// ApplyFromFile creates or updates entities from a given file of response entities.
// In order to work, the create function must return an already exists error as defined in this package.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) ApplyFromFile(from string) (MultiApplyResults[R], error) {
	return a.multiOperation(from, multiOperationApply, true)
}

func (a *GenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, multiOperationApply, p)
}

// DeleteFromFile updates a single entity from a given file containing a response entity.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) DeleteFromFile(from string) (MultiApplyResults[R], error) {
	return a.multiOperation(from, multiOperationDelete, true)
}

func (a *GenericCLI[C, U, R]) DeleteFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, multiOperationDelete, p)
}

type (
	multiOperationName                  string
	multiOperation[C any, U any, R any] interface {
		do(crud CRUD[C, U, R], doc R, results chan MultiApplyResult[R])
	}
	multiOperationCreateImpl[C any, U any, R any] struct{}
	multiOperationUpdateImpl[C any, U any, R any] struct{}
	multiOperationApplyImpl[C any, U any, R any]  struct{}
	multiOperationDeleteImpl[C any, U any, R any] struct{}
)

const (
	multiOperationCreate = "create"
	multiOperationUpdate = "update"
	multiOperationApply  = "apply"
	multiOperationDelete = "delete"
)

func operationFromName[C any, U any, R any](name multiOperationName) (multiOperation[C, U, R], error) {
	switch name {
	case multiOperationCreate:
		return &multiOperationCreateImpl[C, U, R]{}, nil
	case multiOperationUpdate:
		return &multiOperationUpdateImpl[C, U, R]{}, nil
	case multiOperationDelete:
		return &multiOperationDeleteImpl[C, U, R]{}, nil
	case multiOperationApply:
		return &multiOperationApplyImpl[C, U, R]{}, nil
	default:
		return nil, fmt.Errorf("unsupported op: %s", name)
	}
}

func (a *GenericCLI[C, U, R]) multiOperationPrint(from string, opName multiOperationName, p printers.Printer) error {
	if a.bulkPrint {
		var printErr error

		results, err := a.multiOperation(from, opName, true)
		defer func() {
			printErr = p.Print(results.ToList())
		}()
		if err != nil {
			return err
		}

		return printErr
	}

	_, err := a.multiOperation(from, opName, false, func(mar MultiApplyResult[R]) {
		mar.Print(p)
	})
	return err
}

func (a *GenericCLI[C, U, R]) multiOperation(from string, opName multiOperationName, joinErrors bool, callbacks ...func(MultiApplyResult[R])) (results MultiApplyResults[R], err error) {
	var (
		wg         sync.WaitGroup
		once       sync.Once
		resultChan = make(chan MultiApplyResult[R])
	)
	defer once.Do(func() { close(resultChan) })

	wg.Add(1)
	go func() {
		defer wg.Done()
		for result := range resultChan {
			results = append(results, result)
			for _, c := range callbacks {
				c := c
				c(result)
			}
		}
	}()

	docs, err := a.parser.ReadAll(from)
	if err != nil {
		return nil, err
	}

	op, err := operationFromName[C, U, R](opName)
	if err != nil {
		return nil, err
	}

	for index := range docs {
		op.do(a.crud, docs[index], resultChan)
	}

	once.Do(func() { close(resultChan) })

	wg.Wait()

	return results, results.Error(joinErrors)
}

func (m *multiOperationCreateImpl[C, U, R]) do(crud CRUD[C, U, R], doc R, results chan MultiApplyResult[R]) { //nolint:unused
	createDoc, err := crud.ToCreate(doc)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnCreate, Error: fmt.Errorf("error converting to create entity: %w", err)}
		return
	}

	result, err := crud.Create(createDoc)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnCreate, Error: fmt.Errorf("error creating entity: %w", err)}
		return
	}

	results <- MultiApplyResult[R]{Action: MultiApplyCreated, Result: result}
}

func (m *multiOperationUpdateImpl[C, U, R]) do(crud CRUD[C, U, R], doc R, results chan MultiApplyResult[R]) { //nolint:unused
	updateDoc, err := crud.ToUpdate(doc)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnUpdate, Error: fmt.Errorf("error converting to update entity: %w", err)}
		return
	}

	result, err := crud.Update(updateDoc)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnUpdate, Error: fmt.Errorf("error updating entity: %w", err)}
		return
	}

	results <- MultiApplyResult[R]{Action: MultiApplyUpdated, Result: result}
}

func (m *multiOperationApplyImpl[C, U, R]) do(crud CRUD[C, U, R], doc R, results chan MultiApplyResult[R]) { //nolint:unused
	createDoc, err := crud.ToCreate(doc)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnCreate, Error: fmt.Errorf("error converting to create entity: %w", err)}
		return
	}

	result, err := crud.Create(createDoc)
	if err == nil {
		results <- MultiApplyResult[R]{Action: MultiApplyCreated, Result: result}
		return
	}

	if !errors.Is(err, AlreadyExistsError()) {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnCreate, Error: fmt.Errorf("error creating entity: %w", err)}
		return
	}

	update := &multiOperationUpdateImpl[C, U, R]{}
	update.do(crud, doc, results)
}

func (m *multiOperationDeleteImpl[C, U, R]) do(crud CRUD[C, U, R], doc R, results chan MultiApplyResult[R]) { //nolint:unused
	id, err := crud.GetID(doc)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnDelete, Error: fmt.Errorf("error retrieving id from response entity: %w", err)}
		return
	}

	result, err := crud.Delete(id)
	if err != nil {
		results <- MultiApplyResult[R]{Action: MultiApplyErrorOnDelete, Error: fmt.Errorf("error deleting entity: %w", err)}
		return
	}

	results <- MultiApplyResult[R]{Action: MultiApplyDeleted, Result: result}
}
