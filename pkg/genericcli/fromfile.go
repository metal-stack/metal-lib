package genericcli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"gopkg.in/yaml.v3"
)

var alreadyExistsError = errors.New("entity already exists")

func AlreadyExistsError() error {
	return alreadyExistsError
}

const (
	BulkCreated       BulkAction = "created"
	BulkUpdated       BulkAction = "updated"
	BulkDeleted       BulkAction = "deleted"
	BulkErrorOnCreate BulkAction = "error_on_create"
	BulkErrorOnUpdate BulkAction = "error_on_update"
	BulkErrorOnDelete BulkAction = "error_on_delete"
)

type (
	BulkAction string

	BulkResult[R any] struct {
		Result R
		Action BulkAction
		Error  error
	}

	BulkResults[R any] []BulkResult[R]
)

func (m *BulkResult[R]) Print(p printers.Printer) {
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

func (ms BulkResults[R]) ToList() []R {
	var result []R

	for _, m := range ms {
		if m.Error == nil {
			result = append(result, m.Result)
		}
	}

	return result
}

func (ms BulkResults[R]) ToError(joinErrors bool) error {
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
func (a *GenericCLI[C, U, R]) CreateFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[R]{
		from:       from,
		opName:     multiOperationCreate,
		joinErrors: true,
	})
}

func (a *GenericCLI[C, U, R]) CreateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationCreate)
}

// UpdateFromFile updates entities from a given file containing response entities.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) UpdateFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[R]{
		from:       from,
		opName:     multiOperationUpdate,
		joinErrors: true,
	})
}

func (a *GenericCLI[C, U, R]) UpdateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationUpdate)
}

// ApplyFromFile creates or updates entities from a given file of response entities.
// In order to work, the create function must return an already exists error as defined in this package.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) ApplyFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[R]{
		from:       from,
		opName:     multiOperationApply,
		joinErrors: true,
	})
}

func (a *GenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationApply)
}

// DeleteFromFile updates a single entity from a given file containing a response entity.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *GenericCLI[C, U, R]) DeleteFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[R]{
		from:       from,
		opName:     multiOperationDelete,
		joinErrors: true,
	})
}

func (a *GenericCLI[C, U, R]) DeleteFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationDelete)
}

type (
	multiOperationName                  string
	multiOperation[C any, U any, R any] interface {
		do(crud CRUD[C, U, R], doc R) BulkResult[R]
	}
	multiOperationCreateImpl[C any, U any, R any] struct{}
	multiOperationUpdateImpl[C any, U any, R any] struct{}
	multiOperationApplyImpl[C any, U any, R any]  struct{}
	multiOperationDeleteImpl[C any, U any, R any] struct{}

	multiOperationArgs[R any] struct {
		from   string
		opName multiOperationName

		joinErrors bool

		beforeCallbacks    []func(R) error
		afterCallbacks     []func(BulkResult[R]) error
		beforeAllCallbacks []func([]R) error
		afterAllCallbacks  []func(BulkResults[R]) error
	}
)

const (
	multiOperationCreate = "create"
	multiOperationUpdate = "update"
	multiOperationApply  = "apply"
	multiOperationDelete = "delete"
)

func intermediatePrintCallback[R any](p printers.Printer) func(BulkResult[R]) error {
	return func(mar BulkResult[R]) error {
		mar.Print(p)
		return nil
	}
}

func bulkPrintCallback[R any](p printers.Printer) func(BulkResults[R]) error {
	return func(br BulkResults[R]) error {
		return p.Print(br.ToList())
	}
}

func (a *GenericCLI[C, U, R]) securityPromptCallback(c *PromptConfig, opName multiOperationName) func(R) error {
	return func(r R) error {
		id, _, _, err := a.Interface().Convert(r)
		if err != nil {
			return err
		}

		raw, err := yaml.Marshal(r)
		if err != nil {
			return err
		}

		colored := PrintColoredYAML(raw)
		if err != nil {
			return err
		}

		switch opName {
		case multiOperationApply:
			c.Message = fmt.Sprintf("applying %q, continue?\n\n%s\n\n", id, colored)
		case multiOperationCreate:
			c.Message = fmt.Sprintf("creating %q, continue?\n\n%s\n\n", id, colored)
		case multiOperationDelete:
			c.Message = fmt.Sprintf("deleting %q, continue?\n\n%s\n\n", id, colored)
		case multiOperationUpdate:
			c.Message = fmt.Sprintf("updating %q, continue?\n\n%s\n\n", id, colored)
		}

		return PromptCustom(c)
	}
}

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

func (a *GenericCLI[C, U, R]) multiOperationPrint(from string, p printers.Printer, opName multiOperationName) error {
	var beforeCallbacks []func(R) error
	if a.bulkSecurityPrompt != nil {
		beforeCallbacks = append(beforeCallbacks, a.securityPromptCallback(&PromptConfig{
			In:          a.bulkSecurityPrompt.In,
			Out:         a.bulkSecurityPrompt.Out,
			ShowAnswers: true,
		}, opName))
	}

	if a.bulkPrint {
		_, err := a.multiOperation(&multiOperationArgs[R]{
			from:            from,
			opName:          opName,
			joinErrors:      true,
			beforeCallbacks: beforeCallbacks,
			afterAllCallbacks: []func(BulkResults[R]) error{
				bulkPrintCallback[R](p),
			},
		})
		return err
	}

	_, err := a.multiOperation(&multiOperationArgs[R]{
		from:            from,
		opName:          opName,
		joinErrors:      false,
		beforeCallbacks: beforeCallbacks,
		afterCallbacks: []func(mar BulkResult[R]) error{
			intermediatePrintCallback[R](p),
		},
	})
	return err
}

func (a *GenericCLI[C, U, R]) multiOperation(args *multiOperationArgs[R]) (results BulkResults[R], err error) {
	var (
		callbackErr = func(err error) (BulkResults[R], error) {
			bulkErr := results.ToError(args.joinErrors)
			if bulkErr != nil {
				return results, fmt.Errorf("aborting bulk operation: %s, errors already occurred along the way: %w", err.Error(), bulkErr)
			}
			return results, fmt.Errorf("aborting bulk operation: %w", err)
		}
	)

	docs, err := a.parser.ReadAll(args.from)
	if err != nil {
		return nil, err
	}

	for _, c := range args.beforeAllCallbacks {
		c := c
		err := c(docs)
		if err != nil {
			return callbackErr(err)
		}
	}

	op, err := operationFromName[C, U, R](args.opName)
	if err != nil {
		return nil, err
	}

	for index := range docs {
		for _, c := range args.beforeCallbacks {
			c := c
			err := c(docs[index])
			if err != nil {
				return callbackErr(err)
			}
		}

		result := op.do(a.crud, docs[index])

		results = append(results, result)

		for _, c := range args.afterCallbacks {
			c := c
			err := c(result)
			if err != nil {
				return callbackErr(err)
			}
		}
	}

	for _, c := range args.afterAllCallbacks {
		c := c
		err := c(results)
		if err != nil {
			return callbackErr(err)
		}
	}

	return results, results.ToError(args.joinErrors)
}

func (m *multiOperationCreateImpl[C, U, R]) do(crud CRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
	_, createDoc, _, err := crud.Convert(doc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnCreate, Error: fmt.Errorf("error converting to create entity: %w", err)}
	}

	result, err := crud.Create(createDoc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnCreate, Error: fmt.Errorf("error creating entity: %w", err)}
	}

	return BulkResult[R]{Action: BulkCreated, Result: result}
}

func (m *multiOperationUpdateImpl[C, U, R]) do(crud CRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
	_, _, updateDoc, err := crud.Convert(doc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnUpdate, Error: fmt.Errorf("error converting to update entity: %w", err)}
	}

	result, err := crud.Update(updateDoc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnUpdate, Error: fmt.Errorf("error updating entity: %w", err)}
	}

	return BulkResult[R]{Action: BulkUpdated, Result: result}
}

func (m *multiOperationApplyImpl[C, U, R]) do(crud CRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
	_, createDoc, _, err := crud.Convert(doc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnCreate, Error: fmt.Errorf("error converting to create entity: %w", err)}
	}

	result, err := crud.Create(createDoc)
	if err == nil {
		return BulkResult[R]{Action: BulkCreated, Result: result}
	}

	if !errors.Is(err, AlreadyExistsError()) {
		return BulkResult[R]{Action: BulkErrorOnCreate, Error: fmt.Errorf("error creating entity: %w", err)}
	}

	update := &multiOperationUpdateImpl[C, U, R]{}
	return update.do(crud, doc)
}

func (m *multiOperationDeleteImpl[C, U, R]) do(crud CRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
	id, _, _, err := crud.Convert(doc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnDelete, Error: fmt.Errorf("error retrieving id from response entity: %w", err)}
	}

	result, err := crud.Delete(id)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnDelete, Error: fmt.Errorf("error deleting entity: %w", err)}
	}

	return BulkResult[R]{Action: BulkDeleted, Result: result}
}
