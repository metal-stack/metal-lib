package auditing

import (
	"context"
	"log/slog"
	"time"

	"github.com/metal-stack/metal-lib/pkg/pointer"
)

const (
	asyncDefaultBackoff = 200 * time.Millisecond
)

type (
	AsyncConfig struct {
		// AsyncRetry defines the amount of attempts to retry sending an audit trace to a backend in case it failed.
		AsyncRetry int
		// AsyncBackoff defines the backoff after a failed attempt to index an audit trace to a backend.
		AsyncBackoff *time.Duration
	}

	asyncAuditing struct {
		log    *slog.Logger
		config *AsyncConfig
		a      Auditing
	}
)

// NewAsync takes another audit backend and allows indexing audit traces asynchronously.
// If this is used it can occur that audit traces get lost in case the backend is not available for receiving the trace.
// The advantage is that it does not block.
func NewAsync(backend Auditing, log *slog.Logger, ac AsyncConfig) (Auditing, error) {
	a := &asyncAuditing{
		log:    log.WithGroup("auditing").With("audit-backend", "async"),
		config: &ac,
		a:      backend,
	}

	if ac.AsyncBackoff == nil {
		ac.AsyncBackoff = pointer.Pointer(asyncDefaultBackoff)
	}

	a.log.Info("wrapping audit backend in async")

	return a, nil
}

func (a *asyncAuditing) Index(entry Entry) error {
	go func() {
		count := 0

		for {
			err := a.a.Index(entry)
			if err == nil {
				return
			}

			if count > a.config.AsyncRetry {
				a.log.Error("maximum amount of retries reached for sending event to splunk, giving up", "retries", a.config.AsyncRetry, "entry-id", entry.Id, "error", err)
				return
			}

			count++

			a.log.Error("async indexing failed, retrying", "retries", a.config.AsyncRetry, "backoff", a.config.AsyncBackoff.String(), "error", err)
			time.Sleep(*a.config.AsyncBackoff)

			return
		}
	}()

	return nil
}

func (a *asyncAuditing) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	return a.a.Search(ctx, filter)
}
