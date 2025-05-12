package auditing

import (
	"context"
	"fmt"
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
	wrappedBackendType := fmt.Sprintf("%T", backend)

	a := &asyncAuditing{
		log:    log.WithGroup("auditing").With("audit-backend", "async", "wrapped-backend", wrappedBackendType),
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

		log := a.log.With("entry-id", entry.Id, "retries", a.config.AsyncRetry, "backoff", a.config.AsyncBackoff.String())

		for {
			log.Debug("async index", "count", count)

			err := a.a.Index(entry)
			if err == nil {
				return
			}

			if count >= a.config.AsyncRetry {
				log.Error("maximum amount of retries reached for sending event to splunk, giving up", "error", err)
				return
			}

			count++

			log.Error("async indexing failed, retrying", "error", err)
			time.Sleep(*a.config.AsyncBackoff)

			continue
		}
	}()

	return nil
}

func (a *asyncAuditing) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	return a.a.Search(ctx, filter)
}
