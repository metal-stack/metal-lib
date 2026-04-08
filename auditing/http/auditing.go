package http

import (
	"errors"

	"github.com/metal-stack/metal-lib/httperrors"
)

const EntryFilterDefaultLimit int64 = 100

// SerializableError attempts to turn an error into something that is usable for the audit backends.
//
// most errors do not contain public fields (e.g. connect error) and when being serialized will turn into
// an empty map.
//
// some error types (e.g. httperror of this library) can be serialized without any issues, so these
// should stay untouched.
func SerializableError(err error) any {
	if err == nil {
		return nil
	}

	var httpErr *httperrors.HTTPErrorResponse
	if ok := errors.As(err, &httpErr); ok {
		return *httpErr
	}

	// fallback to string (which is better than nothing)
	return struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
}
