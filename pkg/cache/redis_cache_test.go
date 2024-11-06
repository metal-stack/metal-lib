package cache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestRedisCache(t *testing.T) {
	ctx := context.Background()
	s := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: s.Addr()})

	type testObject struct {
		Name string `json:"name"`
	}

	cache := NewRedis(c, "prefix_", 2*time.Second, func(ctx context.Context, key string) (*testObject, error) {
		return &testObject{
			Name: strings.ToUpper(key),
		}, nil
	})

	o, err := cache.Get(ctx, "darth vader")
	require.NoError(t, err)

	assert.Equal(t, &testObject{
		Name: "DARTH VADER",
	}, o)
}
