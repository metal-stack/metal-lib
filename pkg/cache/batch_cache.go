package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type (
	FetchAll[K comparable, O any] func(ctx context.Context) (map[K]O, error)

	FetchAllCache[K comparable, O any] struct {
		expiration time.Duration
		fetchAll   FetchAll[K, O]
		entries    sync.Map
	}
)

func NewFetchAll[K comparable, O any](expiration time.Duration, fetchAll FetchAll[K, O]) *FetchAllCache[K, O] {
	return &FetchAllCache[K, O]{
		expiration: expiration,
		fetchAll:   fetchAll,
		entries:    sync.Map{},
	}
}

func (c *FetchAllCache[K, O]) Get(ctx context.Context, key K) (O, error) {
	v, ok := c.entries.Load(key)
	if !ok {
		return c.fetchForKey(ctx, key)
	}

	entry, ok := v.(*entry[O])
	if !ok {
		c.entries.Delete(key)
		var zero O
		return zero, fmt.Errorf("invalid cache entry, please retry")
	}

	if entry.expired() {
		return c.fetchForKey(ctx, key)
	}

	return entry.value, nil
}

func (c *FetchAllCache[K, O]) fetchForKey(ctx context.Context, key K) (O, error) {
	all, err := c.fetchAll(ctx)
	if err != nil {
		var zero O
		return zero, fmt.Errorf("error fetching cache entry: %w", err)
	}

	for k, v := range all {
		if e, ok := c.entries.Load(k); ok {
			if e, ok := e.(*entry[O]); ok {
				e.update(v, c.expiration)
				continue
			}
		}
		entry := newEntry(v, c.expiration)
		c.entries.Store(k, entry)
	}

	val, ok := all[key]
	if !ok {
		var zero O
		return zero, fmt.Errorf("key not found in cache")
	}

	return val, nil
}
