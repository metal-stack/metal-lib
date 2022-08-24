package multisort

import (
	"fmt"
	"sort"

	"golang.org/x/exp/slices"
)

// Sorter can sort by multiple criteria.
type Sorter[E any] struct {
	defaultSortKeys Keys
	fields          FieldMap[E]
}

// FieldMap defines the fields that the sorter is capable to sort and provides the corresponsing compare funcs.
type FieldMap[E any] map[string]CompareFn[E]

// Key is the key that will be sorted by.
type Key struct {
	ID         string
	Descending bool
}

type Keys []Key

// New creates a new multisorter.
func New[E any](fields FieldMap[E], defaultSortKeys Keys) *Sorter[E] {
	return &Sorter[E]{
		defaultSortKeys: defaultSortKeys,
		fields:          fields,
	}
}

// SortBy sorts the given data by the given sort keys.
func (s *Sorter[E]) SortBy(data []E, keys ...Key) error {
	if len(keys) == 0 {
		keys = s.defaultSortKeys
	}

	if len(keys) == 0 {
		return nil
	}

	err := s.validate(keys...)
	if err != nil {
		return err
	}

	slices.SortStableFunc(data, func(a, b E) bool {
		for _, key := range keys {
			f := s.fields[key.ID]

			switch f(a, b, key.Descending) {
			case Less:
				return true
			case NotEqual:
				return false
			}
		}
		return false
	})

	return nil
}

// AvailableKeys returns the available sort keys that this sorter has been initialized with.
func (s *Sorter[E]) AvailableKeys() []string {
	var res []string
	for k := range s.fields {
		res = append(res, k)
	}

	sort.Strings(res)

	return res
}

func (s *Sorter[E]) validate(keys ...Key) error {
	for _, key := range keys {
		_, ok := s.fields[key.ID]
		if !ok {
			return fmt.Errorf("sort key does not exist: %s", key.ID)
		}
	}
	return nil
}
