package testcommon

import (
	"github.com/google/go-cmp/cmp"
	"gopkg.in/inf.v0"
)

func ErrorStringComparer() cmp.Option {
	return cmp.Comparer(func(x, y error) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil && y != nil {
			return false
		}
		if x != nil && y == nil {
			return false
		}
		return x.Error() == y.Error()
	})
}

func InfDecComparer() cmp.Option {
	return cmp.Comparer(func(x, y *inf.Dec) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil && y != nil {
			return false
		}
		if x != nil && y == nil {
			return false
		}
		return x.Cmp(y) == 0
	})
}
