package genericcli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"gopkg.in/yaml.v3"
)

var errAlreadyExists = errors.New("entity already exists")

func AlreadyExistsError() error {
	return errAlreadyExists
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
		Result   R
		Action   BulkAction
		Error    error
		Duration time.Duration
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
		joined := errors.Join(err, m.Error)
		m.Error = fmt.Errorf("error printing original error, original error: %w", joined)
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
func (a *MultiArgGenericCLI[C, U, R]) CreateFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[C, U, R]{
		from:       from,
		op:         multiOperationCreate[C, U, R]{},
		joinErrors: true,
	})
}

func (a *MultiArgGenericCLI[C, U, R]) CreateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationCreate[C, U, R]{})
}

// UpdateFromFile updates entities from a given file containing response entities.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *MultiArgGenericCLI[C, U, R]) UpdateFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[C, U, R]{
		from:       from,
		op:         multiOperationUpdate[C, U, R]{},
		joinErrors: true,
	})
}

func (a *MultiArgGenericCLI[C, U, R]) UpdateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationUpdate[C, U, R]{})
}

// ApplyFromFile creates or updates entities from a given file of response entities.
// In order to work, the create function must return an already exists error as defined in this package.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *MultiArgGenericCLI[C, U, R]) ApplyFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[C, U, R]{
		from:       from,
		op:         multiOperationApply[C, U, R]{},
		joinErrors: true,
	})
}

func (a *MultiArgGenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationApply[C, U, R]{})
}

// DeleteFromFile updates a single entity from a given file containing a response entity.
//
// As this function uses response entities, it is possible that create and update entity representation
// is inaccurate to a certain degree.
func (a *MultiArgGenericCLI[C, U, R]) DeleteFromFile(from string) (BulkResults[R], error) {
	return a.multiOperation(&multiOperationArgs[C, U, R]{
		from:       from,
		op:         multiOperationDelete[C, U, R]{},
		joinErrors: true,
	})
}

func (a *MultiArgGenericCLI[C, U, R]) DeleteFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiOperationPrint(from, p, multiOperationDelete[C, U, R]{})
}

type (
	multiOperation[C any, U any, R any] interface {
		do(crud MultiArgCRUD[C, U, R], doc R) BulkResult[R]
		verb() string
	}
	multiOperationCreate[C any, U any, R any] struct{}
	multiOperationUpdate[C any, U any, R any] struct{}
	multiOperationApply[C any, U any, R any]  struct{}
	multiOperationDelete[C any, U any, R any] struct{}

	multiOperationArgs[C any, U any, R any] struct {
		from string
		op   multiOperation[C, U, R]

		joinErrors bool

		beforeCallbacks    []func(R) error
		afterCallbacks     []func(BulkResult[R]) error
		beforeAllCallbacks []func([]R) error
		afterAllCallbacks  []func(BulkResults[R]) error
	}
)

func intermediatePrintCallback[R any](p printers.Printer) func(BulkResult[R]) error {
	return func(mar BulkResult[R]) error {
		mar.Print(p)
		return nil
	}
}

func timestampCallback[R any]() func(BulkResult[R]) error {
	return func(mar BulkResult[R]) error {
		fmt.Printf("took %s\n", mar.Duration.String())
		return nil
	}
}

func bulkPrintCallback[R any](p printers.Printer) func(BulkResults[R]) error {
	return func(br BulkResults[R]) error {
		return p.Print(br.ToList())
	}
}

func (a *MultiArgGenericCLI[C, U, R]) securityPromptCallback(c *PromptConfig, op multiOperation[C, U, R]) func(R) error {
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

		c.Message = fmt.Sprintf("%s %q, continue?\n\n%s\n\n", op.verb(), id, colored)

		return PromptCustom(c)
	}
}

func (a *MultiArgGenericCLI[C, U, R]) multiOperationPrint(from string, p printers.Printer, op multiOperation[C, U, R]) error {
	var (
		beforeCallbacks []func(R) error
		afterCallbacks  []func(BulkResult[R]) error
	)

	if a.bulkSecurityPrompt != nil {
		in := a.bulkSecurityPrompt.In
		if in == nil {
			in = os.Stdin
		}
		if f, ok := in.(*os.File); ok {
			if isatty.IsTerminal(f.Fd()) {
				beforeCallbacks = append(beforeCallbacks, a.securityPromptCallback(&PromptConfig{
					In:          a.bulkSecurityPrompt.In,
					Out:         a.bulkSecurityPrompt.Out,
					ShowAnswers: true,
				}, op))
			}
		}
	}

	if a.timestamps {
		afterCallbacks = append(afterCallbacks, timestampCallback[R]())
	}

	if a.bulkPrint {
		_, err := a.multiOperation(&multiOperationArgs[C, U, R]{
			from:            from,
			op:              op,
			joinErrors:      true,
			beforeCallbacks: beforeCallbacks,
			afterCallbacks:  afterCallbacks,
			afterAllCallbacks: []func(BulkResults[R]) error{
				bulkPrintCallback[R](p),
			},
		})
		return err
	}

	_, err := a.multiOperation(&multiOperationArgs[C, U, R]{
		from:            from,
		op:              op,
		joinErrors:      false,
		beforeCallbacks: beforeCallbacks,
		afterCallbacks: append([]func(mar BulkResult[R]) error{
			intermediatePrintCallback[R](p),
		}, afterCallbacks...),
	})
	return err
}

func (a *MultiArgGenericCLI[C, U, R]) multiOperation(args *multiOperationArgs[C, U, R]) (results BulkResults[R], err error) {
	var (
		callbackErr = func(err error) (BulkResults[R], error) {
			bulkErr := results.ToError(args.joinErrors)
			if bulkErr != nil {
				joined := errors.Join(err, bulkErr)
				return results, fmt.Errorf("aborting bulk operation, errors already occurred along the way: %w", joined)
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

	for index := range docs {
		for _, c := range args.beforeCallbacks {
			c := c
			err := c(docs[index])
			if err != nil {
				return callbackErr(err)
			}
		}

		start := time.Now()
		result := args.op.do(a.crud, docs[index])
		result.Duration = time.Since(start)

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

func (m multiOperationCreate[C, U, R]) verb() string { //nolint:unused
	return "creating"
}

func (m multiOperationCreate[C, U, R]) do(crud MultiArgCRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
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

func (m multiOperationUpdate[C, U, R]) verb() string { //nolint:unused
	return "updating"
}

func (m multiOperationUpdate[C, U, R]) do(crud MultiArgCRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
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

func (m multiOperationApply[C, U, R]) verb() string { //nolint:unused
	return "applying"
}

func (m multiOperationApply[C, U, R]) do(crud MultiArgCRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
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

	update := &multiOperationUpdate[C, U, R]{}
	return update.do(crud, doc)
}

func (m multiOperationDelete[C, U, R]) verb() string { //nolint:unused
	return "deleting"
}

func (m multiOperationDelete[C, U, R]) do(crud MultiArgCRUD[C, U, R], doc R) BulkResult[R] { //nolint:unused
	id, _, _, err := crud.Convert(doc)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnDelete, Error: fmt.Errorf("error retrieving id from response entity: %w", err)}
	}

	result, err := crud.Delete(id...)
	if err != nil {
		return BulkResult[R]{Action: BulkErrorOnDelete, Error: fmt.Errorf("error deleting entity: %w", err)}
	}

	return BulkResult[R]{Action: BulkDeleted, Result: result}
}
