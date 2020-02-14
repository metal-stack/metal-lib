package httperrors

import (
	"fmt"
	"net/http"
)

// HTTPErrorResponse is returned in case of functional errors.
type HTTPErrorResponse struct {
	StatusCode int    `json:"statuscode" description:"http status code"`
	Message    string `json:"message" description:"error message"`
}

// FromDefaultResponse creates an error response from a client default http error response. Returns an unconventional error if response could not be unmarshaled.
func FromDefaultResponse(statusCode *int32, message *string, err error) *HTTPErrorResponse {
	if statusCode == nil || message == nil {
		return UnconventionalError(err)
	}

	return &HTTPErrorResponse{
		StatusCode: int(*statusCode),
		Message:    *message,
	}
}

// NewHTTPError creates a new http error.
func NewHTTPError(code int, err error) *HTTPErrorResponse {
	return &HTTPErrorResponse{
		StatusCode: code,
		Message:    err.Error(),
	}
}

// NotFound creates a new notfound error with a given error message. Convenience Method.
func NotFound(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusNotFound, err)
}

// IsNotFound returns true if the error is a not found error
func IsNotFound(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusNotFound
}

// BadRequest creates a new bad request error with a given error message. Convenience Method.
func BadRequest(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusBadRequest, err)
}

// IsBadRequest returns true if the error is a bad request error
func IsBadRequest(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusBadRequest
}

// UnprocessableEntity creates a new unprocessable entity request error with a given error message. Convenience Method.
func UnprocessableEntity(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusUnprocessableEntity, err)
}

// IsUnprocessableEntity returns true if the error is an unprocessable entity error
func IsUnprocessableEntity(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusUnprocessableEntity
}

// Conflict creates a new conflict error with a given error message. Convenience Method.
func Conflict(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusConflict, err)
}

// IsConflict returns true if the error is a conflict error
func IsConflict(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusConflict
}

// Forbidden creates a new forbidden response with a given error message. Convenience Method.
func Forbidden(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusForbidden, err)
}

// IsForbidden returns true if the error is a forbidden error
func IsForbidden(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusForbidden
}

// Unauthorized creates a new unauthorized response with a given error message. Convenience Method.
func Unauthorized(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusUnauthorized, err)
}

// IsUnauthorized returns true if the error is an unauthorized error
func IsUnauthorized(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusUnauthorized
}

// InternalServerError creates a new internal server error with a given error message. Convenience Method.
func InternalServerError(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusInternalServerError, err)
}

// IsInternalServerError returns true if the error is an internal server error
func IsInternalServerError(httperr *HTTPErrorResponse) bool {
	return httperr.StatusCode == http.StatusInternalServerError
}

// UnknownError creates a new internal server error with a given error message. Convenience Method. Is actually also just an internal server error.
func UnknownError(err error) *HTTPErrorResponse {
	return InternalServerError(err)
}

// UnconventionalError creates a new error with a given error message for a response that did not follow the internal error convention. Convenience Method.
func UnconventionalError(err error) *HTTPErrorResponse {
	return NewHTTPError(http.StatusInternalServerError, fmt.Errorf("unexpected error: client response does not follow internal error convention: %v", err.Error()))
}
