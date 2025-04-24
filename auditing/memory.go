package auditing

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"
)

type (
	MemoryConfig struct{}

	memoryAuditing struct {
		component string
		log       *slog.Logger

		memory []Entry
		mutex  sync.RWMutex

		config *MemoryConfig
	}
)

// NewMemory returns a new auditing backend that runs in memory.
// The main intention of this backend is to be used for testing purposes to avoid mocking.
//
// Please note that this backend is not intended to be used for production because it is ephemeral
// and it is not guaranteed to have feature-equality with other auditing backends.
func NewMemory(c Config, tc MemoryConfig) (Auditing, error) {
	if c.Component == "" {
		component, err := defaultComponent()
		if err != nil {
			return nil, err
		}

		c.Component = component
	}

	a := &memoryAuditing{
		component: c.Component,
		log:       c.Log.WithGroup("auditing"),
		memory:    []Entry{},
		config:    &tc,
	}

	a.log.Info("connected to memory backend")

	return a, nil
}

func (a *memoryAuditing) Flush() error {
	return nil
}

func (a *memoryAuditing) Index(entry Entry) error {
	if entry.Component == "" {
		entry.Component = a.component
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.memory = append(a.memory, entry)

	return nil
}

func (a *memoryAuditing) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	var (
		filters []func(e Entry) bool
	)

	if filter.Body != "" {
		filters = append(filters, func(e Entry) bool {
			body, err := json.Marshal(e.Body)
			if err != nil {
				return false
			}

			return strings.Contains(strings.ToLower(string(body)), strings.ToLower(filter.Body))
		})
	}
	if filter.Component != "" {
		filters = append(filters, func(e Entry) bool { return filter.Component == e.Component })
	}
	if filter.Detail != "" {
		filters = append(filters, func(e Entry) bool { return string(filter.Detail) == string(e.Detail) })
	}
	if filter.Error != "" {
		filters = append(filters, func(e Entry) bool {
			if e.Error == nil {
				return false
			}

			if err, ok := e.Error.(error); ok {
				return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(filter.Error))
			}

			errorString, err := json.Marshal(e.Error)
			if err != nil {
				return false
			}

			return strings.Contains(strings.ToLower(string(errorString)), strings.ToLower(filter.Error))
		})
	}
	if filter.ForwardedFor != "" {
		filters = append(filters, func(e Entry) bool { return strings.Contains(e.ForwardedFor, filter.ForwardedFor) })
	}
	if filter.Path != "" {
		filters = append(filters, func(e Entry) bool { return strings.Contains(e.Path, filter.Path) })
	}
	if filter.Phase != "" {
		filters = append(filters, func(e Entry) bool { return string(filter.Phase) == string(e.Phase) })
	}
	if filter.RemoteAddr != "" {
		filters = append(filters, func(e Entry) bool { return strings.Contains(e.RemoteAddr, filter.RemoteAddr) })
	}
	if filter.RequestId != "" {
		filters = append(filters, func(e Entry) bool { return filter.RequestId == e.RequestId })
	}
	if filter.StatusCode != 0 {
		filters = append(filters, func(e Entry) bool { return filter.StatusCode == e.StatusCode })
	}
	if filter.Tenant != "" {
		filters = append(filters, func(e Entry) bool { return filter.Tenant == e.Tenant })
	}
	if filter.Project != "" {
		filters = append(filters, func(e Entry) bool { return filter.Project == e.Project })
	}
	if filter.Type != "" {
		filters = append(filters, func(e Entry) bool { return string(filter.Type) == string(e.Type) })
	}
	if filter.User != "" {
		filters = append(filters, func(e Entry) bool { return filter.User == e.User })
	}

	// to make queries more efficient for memory, we always provide from
	if filter.From.IsZero() {
		filter.From = time.Now().Add(-24 * time.Hour).UTC()
	}

	var entries []Entry

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for _, e := range a.memory {
		if e.Timestamp.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && e.Timestamp.After(filter.To) {
			continue
		}

		match := true
		for _, f := range filters {
			if !f(e) {
				match = false
				break
			}
		}

		if !match {
			continue
		}

		entries = append(entries, e)
	}

	if filter.Limit != 0 && filter.Limit < int64(len(entries)) {
		entries = entries[:filter.Limit]
	}

	return entries, nil
}
