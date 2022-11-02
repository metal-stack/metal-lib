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

	bulkPrint bool
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
	// GetID returns the id from a given response object.
	// This is required for capabilities like delete from file of response objects.
	GetID(r R) (string, error)
}

// NewGenericCLI returns a new generic cli.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
func NewGenericCLI[C any, U any, R any](crud CRUD[C, U, R]) *GenericCLI[C, U, R] {
	fs := afero.NewOsFs()
	return &GenericCLI[C, U, R]{
		crud:      crud,
		fs:        fs,
		parser:    MultiDocumentYAML[R]{fs: fs},
		bulkPrint: true,
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

// WithBulkPrint prints results in a bulk at the end on multi-entity operations, the results are a list.
// default is printing results intermediately during the operation, which causes single entities to be printed in sequence.
func (a *GenericCLI[C, U, R]) WithBulkPrint() *GenericCLI[C, U, R] {
	a.bulkPrint = true
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
		GetID(r *testResponse) (string, error)
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
