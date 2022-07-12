package testcommon

import (
	"context"
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func IgnoreContexts() cmp.Option {
	return cmpopts.IgnoreInterfaces(struct{ context.Context }{})
}

func IgnoreUnexported() cmp.Option {
	// the exporter opt allows all unexported fields: https://github.com/google/go-cmp/pull/176
	return cmp.Exporter(func(reflect.Type) bool { return true })
}
