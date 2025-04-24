package auditing_test

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"testing"

	"github.com/metal-stack/metal-lib/auditing"
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

			auditing, err := auditing.NewMemory(auditing.Config{
				Log: slog.Default(),
			}, auditing.MemoryConfig{})
			require.NoError(t, err)

			tt.t(t, auditing)
		})
	}
}
