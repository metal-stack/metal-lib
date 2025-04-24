package auditing

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditing_Memory(t *testing.T) {
	skips := []string{
		"filter on body missing one word",
		"backwards compatibility with old error type",
	}

	for i, tt := range tests(context.Background()) {
		t.Run(fmt.Sprintf("%d %s", i, tt.name), func(t *testing.T) {
			if slices.Contains(skips, tt.name) {
				t.Skipf("skipping because memory backend does not support this")
			}

			auditing, err := NewMemory(Config{
				Log: slog.Default(),
			}, MemoryConfig{})
			require.NoError(t, err)

			tt.t(t, auditing)
		})
	}
}
