package genericcli

import (
	"io"

	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/spf13/afero"
)

type GenericCLI[C any, U any, R any] struct {
	v2 GenericCLIv2[C, U, R]
}

// CRUD must be implemented in order to get generic CLI functionality.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
type CRUD[C any, U any, R any] interface {
	// Get returns the entity with the given id. It can be that multiple ids are passed in case the id is a compound key.
	Get(id string) (R, error)
	// List returns a slice of entities.
	List() ([]R, error)
	// Create tries to create the entity with the given request and returns the created entity.
	Create(rq C) (R, error)
	// Update tries to update the entity with the given request and returns the updated entity.
	Update(rq U) (R, error)
	// Delete tries to delete the entity with the given id and returns the deleted entity. It can be that multiple ids are passed in case the id is a compound key.
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
	fs := afero.NewOsFs()
	return &GenericCLI[C, U, R]{
		v2: GenericCLIv2[C, U, R]{
			crud:      v2mapper[C, U, R]{},
			fs:        fs,
			parser:    MultiDocumentYAML[R]{fs: fs},
			bulkPrint: false,
		},
	}
}

func (a *GenericCLI[C, U, R]) WithFS(fs afero.Fs) *GenericCLI[C, U, R] {
	a.v2.fs = fs
	a.v2.parser = MultiDocumentYAML[R]{fs: fs}
	return a
}

func (a *GenericCLI[C, U, R]) WithSorter(sorter *multisort.Sorter[R]) *GenericCLI[C, U, R] {
	a.v2.sorter = sorter
	return a
}

// WithBulkPrint prints results in a bulk at the end on multi-entity operations, the results are a list.
// default is printing results intermediately during the bulk operation, which causes single entities to be printed in sequence.
func (a *GenericCLI[C, U, R]) WithBulkPrint() *GenericCLI[C, U, R] {
	a.v2.bulkPrint = true
	return a
}

// WithBulkSecurityPrompt prints interactive prompts before a multi-entity operation if there is a tty.
func (a *GenericCLI[C, U, R]) WithBulkSecurityPrompt(in io.Reader, out io.Writer) *GenericCLI[C, U, R] {
	a.v2.bulkSecurityPrompt = &PromptConfig{
		In:  in,
		Out: out,
	}
	return a
}

// WithBulkTimestamps prints out the duration of an operation to stdout during a bulk operation.
func (a *GenericCLI[C, U, R]) WithTimestamps() *GenericCLI[C, U, R] {
	a.v2.timestamps = true
	return a
}

// Interface returns the interface that was used to create this generic cli.
func (a *GenericCLI[C, U, R]) Interface() CRUDv2[C, U, R] {
	return a.v2.crud
}

// Sorter returns the sorter of this generic cli.
func (a *GenericCLI[C, U, R]) Sorter() *multisort.Sorter[R] {
	return a.v2.sorter
}

// following only used for mock generation (has to be in non-test file), do not use:

type (
	testClient interface {
		Get(id ...string) (*testResponse, error)
		List() ([]*testResponse, error)
		Create(rq *testCreate) (*testResponse, error)
		Update(rq *testUpdate) (*testResponse, error)
		Delete(id ...string) (*testResponse, error)
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

type v2mapper[C any, U any, R any] struct {
	v1 CRUD[C, U, R]
}

func (v v2mapper[C, U, R]) Get(ids ...string) (R, error) {
	id, err := GetExactlyOneArg(ids)
	if err != nil {
		var zero R
		return zero, err
	}

	return v.v1.Get(id)
}

func (v v2mapper[C, U, R]) List() ([]R, error) {
	return v.v1.List()
}

func (v v2mapper[C, U, R]) Create(rq C) (R, error) {
	return v.v1.Create(rq)
}

func (v v2mapper[C, U, R]) Update(rq U) (R, error) {
	return v.v1.Update(rq)
}

func (v v2mapper[C, U, R]) Delete(ids ...string) (R, error) {
	id, err := GetExactlyOneArg(ids)
	if err != nil {
		var zero R
		return zero, err
	}

	return v.v1.Delete(id)
}

func (v v2mapper[C, U, R]) Convert(r R) ([]string, C, U, error) {
	id, cr, ur, err := v.v1.Convert(r)
	return []string{id}, cr, ur, err
}
