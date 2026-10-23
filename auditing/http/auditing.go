package http

import (
	"github.com/metal-stack/metal-lib/auditing"
)

const EntryFilterDefaultLimit int64 = 100

// SerializableError attempts to turn an error into something that is usable for the audit backends.
//
// most errors do not contain public fields (e.g. connect error) and when being serialized will turn into
// an empty map.
//
// some error types (e.g. httperror of this library) can be serialized without any issues, so these
// should stay untouched.
//
// Deprecated: use auditing.SerializableError(auditing.EntryTypeHTTP, err).
func SerializableError(err error) any {
	return auditing.SerializableError(auditing.EntryTypeHTTP, err)
}
