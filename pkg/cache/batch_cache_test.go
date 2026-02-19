package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"github.com/stretchr/testify/assert"
)

func Test_FetchAllCache(t *testing.T) {
	type TestElement struct {
		ID string
	}
	type TestResponse map[string]TestElement

	tests := []struct {
		name       string
		times      int
		expiration time.Duration
		delay      time.Duration
		response   TestResponse
		want       TestElement
		wantCount  int
		wantErr    error
	}{
		{
			name:       "only called once",
			response:   TestResponse{"1": {ID: "1"}},
			want:       TestElement{ID: "1"},
			times:      100,
			expiration: 1 * time.Second,
			delay:      1 * time.Millisecond,
			wantCount:  1,
			wantErr:    nil,
		},
		{
			name:       "called twice",
			response:   TestResponse{"1": {ID: "1"}},
			want:       TestElement{ID: "1"},
			times:      4,
			expiration: 1 * time.Second,
			delay:      400 * time.Millisecond,
			wantCount:  2,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			cache := NewFetchAll(tt.expiration, func(ctx context.Context) (map[string]TestElement, error) {
				count++
				return tt.response, nil
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
