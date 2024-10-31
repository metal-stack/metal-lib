package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type (
	RedisCache[O any] struct {
		client     *redis.Client
		expiration time.Duration
		fetch      FetchFunc[string, O]
		prefix     string
	}
)

func NewRedis[O any](client *redis.Client, prefix string, expiration time.Duration, fetch FetchFunc[string, O]) *RedisCache[O] {
	return &RedisCache[O]{
		client:     client,
		expiration: expiration,
		fetch:      fetch,
		prefix:     prefix,
	}
}

func (c *RedisCache[O]) prefixedKey(key string) string {
	return c.prefix + key
}

func (c *RedisCache[O]) Get(ctx context.Context, key string) (O, error) {
	var o O

	encoded, err := c.client.Get(ctx, c.prefixedKey(key)).Result()
	if err == nil {
		err = json.Unmarshal([]byte(encoded), &o)
		if err != nil {
			return o, err
		}

		return o, nil
	}

	if !errors.Is(err, redis.Nil) {
		return o, err
	}

	o, err = c.fetch(ctx, key)
	if err != nil {
		return o, fmt.Errorf("error fetching cache entry: %w", err)
	}

	enc, err := json.Marshal(o)
	if err != nil {
		return o, fmt.Errorf("unable to marshal cache entry: %w", err)
	}

	_, err = c.client.Set(ctx, c.prefixedKey(key), enc, c.expiration).Result()
	if err != nil {
		return o, fmt.Errorf("unable to store cache entry: %w", err)
	}

	return o, nil
}

// Refresh the entry with given key regardless expiration
func (c *RedisCache[O]) Refresh(ctx context.Context, key string) (O, error) {
	o, err := c.fetch(ctx, key)
	if err != nil {
		var zero O
		return zero, fmt.Errorf("error fetching cache entry: %w", err)
	}

	enc, err := json.Marshal(o)
	if err != nil {
		return o, fmt.Errorf("unable to marshal cache entry: %w", err)
	}

	_, err = c.client.Set(ctx, c.prefixedKey(key), enc, c.expiration).Result()
	if err != nil {
		return o, fmt.Errorf("unable to store cache entry: %w", err)
	}

	return o, nil
}
