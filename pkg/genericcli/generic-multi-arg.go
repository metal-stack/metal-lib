package genericcli

import (
	"io"

	"github.com/metal-stack/metal-lib/pkg/multisort"
	"github.com/spf13/afero"
)

// MultiArgGenericCLI can be used to gain generic CLI functionality.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
type MultiArgGenericCLI[C any, U any, R any] struct {
	fs     afero.Fs
	crud   MultiArgCRUD[C, U, R]
	parser MultiDocumentYAML[R]
	sorter *multisort.Sorter[R]

	bulkPrint          bool
	bulkSecurityPrompt *PromptConfig
	timestamps         bool
}

// MultiArgCRUD must be implemented in order to get generic CLI functionality.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
type MultiArgCRUD[C any, U any, R any] interface {
	// Get returns the entity with the given id. It can be that multiple ids are passed in case the id is a compound key.
	Get(id ...string) (R, error)
	// List returns a slice of entities.
	List() ([]R, error)
	// Create tries to create the entity with the given request and returns the created entity.
	Create(rq C) (R, error)
	// Update tries to update the entity with the given request and returns the updated entity.
	Update(rq U) (R, error)
	// Delete tries to delete the entity with the given id and returns the deleted entity. It can be that multiple ids are passed in case the id is a compound key.
	Delete(id ...string) (R, error)
	// Convert converts an entity's response object to best possible create and update requests and additionally returns the entities ID.
	// This is required for capabilities like creation/update/deletion from a file of response objects.
	Convert(r R) ([]string, C, U, error)
}

// NewGenericMultiArgCLI returns a new generic cli.
//
// C is the create request for an entity.
// U is the update request for an entity.
// R is the response object of an entity.
func NewGenericMultiArgCLI[C any, U any, R any](crud CRUD[C, U, R]) *MultiArgGenericCLI[C, U, R] {
	fs := afero.NewOsFs()
	return &MultiArgGenericCLI[C, U, R]{
		crud:      multiArgMapper[C, U, R]{},
		fs:        fs,
		parser:    MultiDocumentYAML[R]{fs: fs},
		bulkPrint: false,
	}
}

func (a *MultiArgGenericCLI[C, U, R]) WithFS(fs afero.Fs) *MultiArgGenericCLI[C, U, R] {
	a.fs = fs
	a.parser = MultiDocumentYAML[R]{fs: fs}
	return a
}

func (a *MultiArgGenericCLI[C, U, R]) WithSorter(sorter *multisort.Sorter[R]) *MultiArgGenericCLI[C, U, R] {
	a.sorter = sorter
	return a
}

// WithBulkPrint prints results in a bulk at the end on multi-entity operations, the results are a list.
// default is printing results intermediately during the bulk operation, which causes single entities to be printed in sequence.
func (a *MultiArgGenericCLI[C, U, R]) WithBulkPrint() *MultiArgGenericCLI[C, U, R] {
	a.bulkPrint = true
	return a
}

// WithBulkSecurityPrompt prints interactive prompts before a multi-entity operation if there is a tty.
func (a *MultiArgGenericCLI[C, U, R]) WithBulkSecurityPrompt(in io.Reader, out io.Writer) *MultiArgGenericCLI[C, U, R] {
	a.bulkSecurityPrompt = &PromptConfig{
		In:  in,
		Out: out,
	}
	return a
}

// WithBulkTimestamps prints out the duration of an operation to stdout during a bulk operation.
func (a *MultiArgGenericCLI[C, U, R]) WithTimestamps() *MultiArgGenericCLI[C, U, R] {
	a.timestamps = true
	return a
}

// Interface returns the interface that was used to create this generic cli.
func (a *MultiArgGenericCLI[C, U, R]) Interface() MultiArgCRUD[C, U, R] {
	return a.crud
}

// Sorter returns the sorter of this generic cli.
func (a *MultiArgGenericCLI[C, U, R]) Sorter() *multisort.Sorter[R] {
	return a.sorter
}
