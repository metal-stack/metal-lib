package testcommon

import (
	"time"

	"github.com/go-openapi/strfmt"
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

func StrFmtDateComparer() cmp.Option {
	return cmp.Comparer(func(x, y strfmt.DateTime) bool {
		return time.Time(x).Unix() == time.Time(y).Unix()
	})
}
