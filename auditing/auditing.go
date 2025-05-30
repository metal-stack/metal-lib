package auditing

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Component string
	Log       *slog.Logger
	// IndexTimeout sets a timeout for indexing a trace for the backend.
	IndexTimeout time.Duration
}

type Interval string

var (
	HourlyInterval  Interval = "@hourly"
	DailyInterval   Interval = "@daily"
	MonthlyInterval Interval = "@monthly"
)

type EntryType string

const (
	EntryTypeHTTP  EntryType = "http"
	EntryTypeGRPC  EntryType = "grpc"
	EntryTypeEvent EntryType = "event"
)

type EntryDetail string

const (
	EntryDetailGRPCUnary  EntryDetail = "unary"
	EntryDetailGRPCStream EntryDetail = "stream"
)

type EntryPhase string

const (
	EntryPhaseRequest  EntryPhase = "request"
	EntryPhaseResponse EntryPhase = "response"
	EntryPhaseSingle   EntryPhase = "single"
	EntryPhaseError    EntryPhase = "error"
	EntryPhaseOpened   EntryPhase = "opened"
	EntryPhaseClosed   EntryPhase = "closed"
)

const EntryFilterDefaultLimit int64 = 100

type Entry struct {
	Id        string    `json:"-"` // filled by the auditing driver
	Component string    `json:"component"`
	RequestId string    `json:"rqid"`
	Type      EntryType `json:"type"`
	Timestamp time.Time `json:"timestamp"`

	User    string `json:"user"`
	Tenant  string `json:"tenant"`
	Project string `json:"project"`

	// For `EntryDetailHTTP` the HTTP method get, post, put, delete, ...
	// For `EntryDetailGRPC` unary, stream
	Detail EntryDetail `json:"detail"`
	// e.g. Request, Response, Error, Opened, Close
	Phase EntryPhase `json:"phase"`
	// For `EntryDetailHTTP` /api/v1/...
	// For `EntryDetailGRPC` /api.v1/... (the method name)
	Path         string `json:"path"`
	ForwardedFor string `json:"forwardedfor"`
	RemoteAddr   string `json:"remoteaddr"`

	Body       any  `json:"body"`       // JSON, string or numbers
	StatusCode *int `json:"statuscode"` // for `EntryDetailHTTP` the HTTP status code, for EntryDetailGRPC` the grpc status code

	// Internal errors
	Error any `json:"error"`
}

func (e *Entry) prepareForNextPhase() {
	e.Id = ""
	e.Timestamp = time.Now()
	e.Body = nil
	e.Error = nil

	switch e.Phase {
	case EntryPhaseRequest:
		e.Phase = EntryPhaseResponse
	case EntryPhaseOpened:
		e.Phase = EntryPhaseClosed
	case EntryPhaseResponse,
		EntryPhaseSingle,
		EntryPhaseError,
		EntryPhaseClosed:
		// keep the phase
	}
}

type EntryFilter struct {
	Limit int64 `json:"limit" optional:"true"` // default `EntryFilterDefaultLimit`

	// In range
	From time.Time `json:"from" optional:"true"`
	To   time.Time `json:"to" optional:"true"`

	Component string    `json:"component" optional:"true"` // exact match
	RequestId string    `json:"rqid" optional:"true"`      // starts with
	Type      EntryType `json:"type" optional:"true"`      // exact match

	User    string `json:"user" optional:"true"`    // exact match
	Tenant  string `json:"tenant" optional:"true"`  // exact match
	Project string `json:"project" optional:"true"` // exact match

	Detail EntryDetail `json:"detail" optional:"true"` // exact match
	Phase  EntryPhase  `json:"phase" optional:"true"`  // exact match

	Path         string `json:"path" optional:"true"`          // free text
	ForwardedFor string `json:"forwarded_for" optional:"true"` // free text
	RemoteAddr   string `json:"remote_addr" optional:"true"`   // free text

	Body       string `json:"body" optional:"true"`        // free text
	StatusCode *int   `json:"status_code" optional:"true"` // exact match

	Error string `json:"error" optional:"true"` // free text
}

type Auditing interface {
	// Adds the given entry to the index.
	// Some fields like `Id`, `Component` and `Timestamp` will be filled by the auditing driver if not given.
	Index(Entry) error
	// Searches for entries matching the given filter.
	// By default only recent entries will be returned.
	// The returned entries will be sorted by timestamp in descending order.
	Search(context.Context, EntryFilter) ([]Entry, error)
}

func defaultComponent() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Base(ex), nil
}
