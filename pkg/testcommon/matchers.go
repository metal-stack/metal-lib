package testcommon

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/mock"
)

func MatchByCmpDiff(t *testing.T, i interface{}, opts ...cmp.Option) interface{} {
	opts = append(opts, IgnoreUnexported())
	return mock.MatchedBy(func(j interface{}) bool {
		diff := cmp.Diff(i, j, opts...)
		if diff != "" {
			if t != nil {
				t.Log(diff)
			}
			return false
		}
		return true
	})
}

// MatchIgnoreContext is a mock matcher that ignores contexts inside the given object.
//
// interfaces generated by swagger bury the ctx into the params, which makes it impossible to mock
// with this matcher you can use the mocks quite adequately.
func MatchIgnoreContext(t *testing.T, i interface{}) interface{} {
	return mock.MatchedBy(func(j interface{}) bool {
		// the exporter opt allows all unexported fields: https://github.com/google/go-cmp/pull/176
		diff := cmp.Diff(i, j, IgnoreContexts(), IgnoreUnexported())
		if diff != "" {
			if t != nil {
				t.Log(diff)
			}
			return false
		}
		return true
	})
}

func IgnoreContexts() cmp.Option {
	return cmpopts.IgnoreInterfaces(struct{ context.Context }{})
}

func IgnoreUnexported() cmp.Option {
	// the exporter opt allows all unexported fields: https://github.com/google/go-cmp/pull/176
	return cmp.Exporter(func(reflect.Type) bool { return true })
}
