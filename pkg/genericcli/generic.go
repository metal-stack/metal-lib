package genericcli

import (
	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/spf13/afero"
)

// GenericCLI can be used to gain generic CLI functionality.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
type GenericCLI[C any, U any, R any] struct {
	fs     afero.Fs
	crud   CRUD[C, U, R]
	parser MultiDocumentYAML[R]
	sorter *multisort.Sorter[R]

	intermediateApplyPrint bool
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
	// ToCreate transforms an entity's response object to its create request.
	// This is required for capabilities like creation from file of response objects.
	ToCreate(r R) (C, error)
	// ToUpdate transforms an entity's response object to its update request.
	// This is required for capabilities like update from file of response objects or edit.
	ToUpdate(r R) (U, error)
}

// NewGenericCLI returns a new generic cli.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
func NewGenericCLI[C any, U any, R any](crud CRUD[C, U, R]) *GenericCLI[C, U, R] {
	fs := afero.NewOsFs()
	return &GenericCLI[C, U, R]{
		crud:                   crud,
		fs:                     fs,
		parser:                 MultiDocumentYAML[R]{fs: fs},
		intermediateApplyPrint: true,
	}
}

func (a *GenericCLI[C, U, R]) WithFS(fs afero.Fs) *GenericCLI[C, U, R] {
	a.fs = fs
	a.parser = MultiDocumentYAML[R]{fs: fs}
	return a
}

func (a *GenericCLI[C, U, R]) WithSorter(sorter *multisort.Sorter[R]) *GenericCLI[C, U, R] {
	a.sorter = sorter
	return a
}

// WithApplyBulkPrint prints apply results in a bulk at the end, the results are a list.
// default is printing results intermediately during apply, which causes single entities to be printed sequentially.
func (a *GenericCLI[C, U, R]) WithApplyBulkPrint() *GenericCLI[C, U, R] {
	a.intermediateApplyPrint = false
	return a
}

// Interface returns the interface that was used to create this generic cli.
func (a *GenericCLI[C, U, R]) Interface() CRUD[C, U, R] {
	return a.crud
}

// Sorter returns the sorter of this generic cli.
func (a *GenericCLI[C, U, R]) Sorter() *multisort.Sorter[R] {
	return a.sorter
}

// following only used for mock generation (has to be in non-test file), do not use:

type (
	testClient interface {
		Get(id string) (*testResponse, error)
		List() ([]*testResponse, error)
		Create(rq *testCreate) (*testResponse, error)
		Update(rq *testUpdate) (*testResponse, error)
		Delete(id string) (*testResponse, error)
		ToCreate(r *testResponse) (*testCreate, error)
		ToUpdate(r *testResponse) (*testUpdate, error)
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
