package genericcli

import (
	"io"

	"github.com/metal-stack/metal-lib/pkg/genericcli/printers"
	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/spf13/afero"
)

// GenericCLI can be used to gain generic CLI functionality.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
type GenericCLI[C any, U any, R any] struct {
	// internally we map everyhing to multi arg cli to reduce code redundance
	multiCLI *MultiArgGenericCLI[C, U, R]
}

// CRUD must be implemented in order to get generic CLI functionality.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
type CRUD[C any, U any, R any] interface {
	// Get returns the entity with the given id.
	Get(id string) (R, error)
	// List returns a slice of entities.
	List() ([]R, error)
	// Create tries to create the entity with the given request and returns the created entity.
	Create(rq C) (R, error)
	// Update tries to update the entity with the given request and returns the updated entity.
	Update(rq U) (R, error)
	// Delete tries to delete the entity with the given id and returns the deleted entity.
	Delete(id string) (R, error)
	// Convert converts an entity's response object to best possible create and update requests and additionally returns the entities ID.
	// This is required for capabilities like creation/update/deletion from a file of response objects.
	Convert(r R) (string, C, U, error)
}

// NewGenericCLI returns a new generic cli.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
func NewGenericCLI[C any, U any, R any](crud CRUD[C, U, R]) *GenericCLI[C, U, R] {
	return &GenericCLI[C, U, R]{
		multiCLI: NewGenericMultiArgCLI(multiArgMapper[C, U, R]{singleArg: crud}),
	}
}

func (a *GenericCLI[C, U, R]) WithFS(fs afero.Fs) *GenericCLI[C, U, R] {
	a.multiCLI.WithFS(fs)
	return a
}

func (a *GenericCLI[C, U, R]) WithSorter(sorter *multisort.Sorter[R]) *GenericCLI[C, U, R] {
	a.multiCLI.WithSorter(sorter)
	return a
}

// WithBulkPrint prints results in a bulk at the end on multi-entity operations, the results are a list.
// default is printing results intermediately during the bulk operation, which causes single entities to be printed in sequence.
func (a *GenericCLI[C, U, R]) WithBulkPrint() *GenericCLI[C, U, R] {
	a.multiCLI.WithBulkPrint()
	return a
}

// WithBulkSecurityPrompt prints interactive prompts before a multi-entity operation if there is a tty.
func (a *GenericCLI[C, U, R]) WithBulkSecurityPrompt(in io.Reader, out io.Writer) *GenericCLI[C, U, R] {
	a.multiCLI.WithBulkSecurityPrompt(in, out)
	return a
}

// WithBulkTimestamps prints out the duration of an operation to stdout during a bulk operation.
func (a *GenericCLI[C, U, R]) WithTimestamps() *GenericCLI[C, U, R] {
	a.multiCLI.WithTimestamps()
	return a
}

// Interface returns the interface that was used to create this generic cli.
func (a *GenericCLI[C, U, R]) Interface() MultiArgCRUD[C, U, R] {
	return a.multiCLI.Interface()
}

// Sorter returns the sorter of this generic cli.
func (a *GenericCLI[C, U, R]) Sorter() *multisort.Sorter[R] {
	return a.multiCLI.Sorter()
}

func (a *GenericCLI[C, U, R]) List(sortKeys ...multisort.Key) ([]R, error) {
	return a.multiCLI.List(sortKeys...)
}
func (a *GenericCLI[C, U, R]) ListAndPrint(p printers.Printer, sortKeys ...multisort.Key) error {
	return a.multiCLI.ListAndPrint(p, sortKeys...)
}
func (a *GenericCLI[C, U, R]) Describe(id string) (R, error) {
	return a.multiCLI.Describe(id)
}
func (a *GenericCLI[C, U, R]) DescribeAndPrint(id string, p printers.Printer) error {
	return a.multiCLI.DescribeAndPrint(p, id)
}
func (a *GenericCLI[C, U, R]) Delete(id string) (R, error) {
	return a.multiCLI.Delete(id)
}
func (a *GenericCLI[C, U, R]) DeleteAndPrint(id string, p printers.Printer) error {
	return a.multiCLI.DeleteAndPrint(p, id)
}
func (a *GenericCLI[C, U, R]) Create(rq C) (R, error) {
	return a.multiCLI.Create(rq)
}
func (a *GenericCLI[C, U, R]) CreateAndPrint(rq C, p printers.Printer) error {
	return a.multiCLI.CreateAndPrint(rq, p)
}
func (a *GenericCLI[C, U, R]) Update(rq U) (R, error) {
	return a.multiCLI.Update(rq)
}
func (a *GenericCLI[C, U, R]) UpdateAndPrint(rq U, p printers.Printer) error {
	return a.multiCLI.UpdateAndPrint(rq, p)
}
func (a *GenericCLI[C, U, R]) Edit(args []string) (R, error) {
	return a.multiCLI.Edit(1, args)
}
func (a *GenericCLI[C, U, R]) EditAndPrint(args []string, p printers.Printer) error {
	return a.multiCLI.EditAndPrint(1, args, p)
}
func (a *GenericCLI[C, U, R]) CreateFromFile(from string) (BulkResults[R], error) {
	return a.multiCLI.CreateFromFile(from)
}
func (a *GenericCLI[C, U, R]) CreateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiCLI.CreateFromFileAndPrint(from, p)
}
func (a *GenericCLI[C, U, R]) UpdateFromFile(from string) (BulkResults[R], error) {
	return a.multiCLI.UpdateFromFile(from)
}
func (a *GenericCLI[C, U, R]) UpdateFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiCLI.UpdateFromFileAndPrint(from, p)
}
func (a *GenericCLI[C, U, R]) ApplyFromFile(from string) (BulkResults[R], error) {
	return a.multiCLI.ApplyFromFile(from)
}
func (a *GenericCLI[C, U, R]) ApplyFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiCLI.ApplyFromFileAndPrint(from, p)
}
func (a *GenericCLI[C, U, R]) DeleteFromFile(from string) (BulkResults[R], error) {
	return a.multiCLI.DeleteFromFile(from)
}
func (a *GenericCLI[C, U, R]) DeleteFromFileAndPrint(from string, p printers.Printer) error {
	return a.multiCLI.DeleteFromFileAndPrint(from, p)
}

type multiArgMapper[C any, U any, R any] struct {
	singleArg CRUD[C, U, R]
}

func (v multiArgMapper[C, U, R]) Get(ids ...string) (R, error) {
	id, err := GetExactlyOneArg(ids)
	if err != nil {
		var zero R
		return zero, err
	}

	return v.singleArg.Get(id)
}

func (v multiArgMapper[C, U, R]) List() ([]R, error) {
	return v.singleArg.List()
}

func (v multiArgMapper[C, U, R]) Create(rq C) (R, error) {
	return v.singleArg.Create(rq)
}

func (v multiArgMapper[C, U, R]) Update(rq U) (R, error) {
	return v.singleArg.Update(rq)
}

func (v multiArgMapper[C, U, R]) Delete(ids ...string) (R, error) {
	id, err := GetExactlyOneArg(ids)
	if err != nil {
		var zero R
		return zero, err
	}

	return v.singleArg.Delete(id)
}

func (v multiArgMapper[C, U, R]) Convert(r R) ([]string, C, U, error) {
	id, cr, ur, err := v.singleArg.Convert(r)
	return []string{id}, cr, ur, err
}

// following only used for mock generation (has to be in non-test file), do not use:

type (
	testClient interface {
		Get(id string) (*testResponse, error)
		List() ([]*testResponse, error)
		Create(rq *testCreate) (*testResponse, error)
		Update(rq *testUpdate) (*testResponse, error)
		Delete(id string) (*testResponse, error)
		Convert(r *testResponse) ([]string, *testCreate, *testUpdate, error)
	}
	testCRUD   struct{ client testClient }
	testCreate struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	testUpdate struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	testResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
)
