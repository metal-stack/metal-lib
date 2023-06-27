package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Cache(t *testing.T) {
	type TestResponse struct {
		ID string
	}

	tests := []struct {
		name       string
		times      int
		expiration time.Duration
		delay      time.Duration
		want       *TestResponse
		wantCount  int
		wantErr    error
	}{
		{
			name:       "only called once",
			want:       &TestResponse{ID: "1"},
			times:      100,
			expiration: 1 * time.Second,
			delay:      1 * time.Millisecond,
			wantCount:  1,
			wantErr:    nil,
		},
		{
			name:       "called twice",
			want:       &TestResponse{ID: "1"},
			times:      4,
			expiration: 1 * time.Second,
			delay:      400 * time.Millisecond,
			wantCount:  2,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			cache := New(tt.expiration, func(_ context.Context, key string) (*TestResponse, error) {
				require.Equal(t, key, tt.want.ID, "unexpected key passed")
				count++
				return tt.want, nil
			})

			for i := 0; i < tt.times; i++ {
				got, err := cache.Get(context.Background(), tt.want.ID)
				if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
					t.Errorf("error diff (+got -want):\n %s", diff)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("diff (+got -want):\n %s", diff)
				}
				time.Sleep(tt.delay)
			}

			assert.Equal(t, tt.wantCount, count)
		})
	}
}
