package auditing

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_asyncAuditing_Index(t *testing.T) {
	tests := []struct {
		name         string
		asyncRetry   int
		asyncBackoff time.Duration

		idxFn func(count int) error

		wantCount   int
		wantTimeout bool
	}{
		{
			name:         "index without error",
			asyncRetry:   0,
			asyncBackoff: 5 * time.Millisecond,
			idxFn: func(_ int) error {
				return nil
			},
			wantCount:   0,
			wantTimeout: false,
		},
		{
			name:         "index with error",
			asyncRetry:   0,
			asyncBackoff: 5 * time.Millisecond,
			idxFn: func(_ int) error {
				return errors.New("test backend error")
			},
			wantCount:   1,
			wantTimeout: true,
		},
		{
			name:         "retry does work",
			asyncRetry:   3,
			asyncBackoff: 5 * time.Millisecond,
			idxFn: func(count int) error {
				switch count {
				case 0, 1, 2:
					return errors.New("test backend error")
				default:
					return nil
				}
			},
			wantCount:   3,
			wantTimeout: false,
		},
		{
			name:         "giving up on too many retries",
			asyncRetry:   3,
			asyncBackoff: 5 * time.Millisecond,
			idxFn: func(count int) error {
				switch count {
				case 0, 1, 2, 3:
					return errors.New("test backend error")
				default:
					return nil
				}
			},
			wantCount:   4,
			wantTimeout: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan bool)
			defer close(done)

			backend := &testBackend{idxFn: tt.idxFn, done: done}

			async, err := NewAsync(backend, slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})), AsyncConfig{
				AsyncRetry:   tt.asyncRetry,
				AsyncBackoff: &tt.asyncBackoff,
			})
			require.NoError(t, err)

			err = async.Index(Entry{Id: "test"})
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			select {
			case <-done:
				require.False(t, tt.wantTimeout, "finished but timeout was expected")
			case <-ctx.Done():
				require.True(t, tt.wantTimeout, "unexpected timeout occurred")
			}

			backend.mutex.Lock()
			defer backend.mutex.Unlock()

			assert.Equal(t, tt.wantCount, backend.count)
		})
	}
}

type testBackend struct {
	mutex sync.Mutex
	done  chan bool
	count int
	idxFn func(count int) error
}

func (t *testBackend) Index(e Entry) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.idxFn != nil {
		if err := t.idxFn(t.count); err != nil {
			t.count++
			return errors.New("test backend error")
		}
	}

	t.done <- true

	return nil
}

func (t *testBackend) Search(ctx context.Context, filter EntryFilter) ([]Entry, error) {
	panic("not required")
}
