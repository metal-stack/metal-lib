package sorters

import (
	"github.com/metal-stack/metal-lib/pkg/commands/types"
	"github.com/metal-stack/metal-lib/pkg/multisort"
)

func ContextSorter() *multisort.Sorter[*types.Context] {
	return multisort.New(multisort.FieldMap[*types.Context]{
		"name": func(a, b *types.Context, descending bool) multisort.CompareResult {
			return multisort.Compare(a.Name, b.Name, descending)
		},
	}, multisort.Keys{{ID: "name"}})
}
