package auditing

import (
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Component        string
	URL              string
	APIKey           string
	IndexPrefix      string
	RotationInterval Interval
	Keep             int64
	Log              *zap.SugaredLogger
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
	EntryPhaseRequest  = "request"
	EntryPhaseResponse = "response"
	EntryPhaseSingle   = "single"
	EntryPhaseError    = "error"
	EntryPhaseOpened   = "opened"
	EntryPhaseClosed   = "closed"
)

type Entry struct {
	Id        string // filled by the auditing driver
	Component string
	RequestId string `json:"rqid"`
	Type      EntryType
	Timestamp time.Time

	User   string
	Tenant string

	// For `EntryDetailHTTP` the HTTP method get, post, put, delete, ...
	// For `EntryDetailGRPC` unary, stream
	Detail EntryDetail
	// e.g. Request, Response, Error, Opened, Close
	Phase EntryPhase
	// For `EntryDetailHTTP` /api/v1/...
	// For `EntryDetailGRPC` /api.v1/... (the method name)
	Path         string
	ForwardedFor string
	RemoteAddr   string

	Body       any // JSON, string or numbers
	StatusCode int // only for `EntryDetailHTTP`

	// Internal errors
	Error error
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
	}
}

type Auditing interface {
	Index(Entry) error
}
