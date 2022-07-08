package genericcli

import (
	"github.com/spf13/afero"
)

// GenericCLI can be used to gain generic CLI functionality.
type GenericCLI[C any, U any, R any] struct {
	fs   afero.Fs
	crud CRUD[C, U, R]
}

// CRUD must be implemented in order to get generic CLI functionality.
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
}

// NewGenericCLI returns a new generic cli.
func NewGenericCLI[C any, U any, R any](crud CRUD[C, U, R]) *GenericCLI[C, U, R] {
	return &GenericCLI[C, U, R]{
		crud: crud,
		fs:   afero.NewOsFs(),
	}
}

// Interface returns the interface that was used to create this generic cli.
func (a *GenericCLI[C, U, R]) Interface() CRUD[C, U, R] {
	return a.crud
}
