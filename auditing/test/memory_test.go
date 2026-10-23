package test

import (
	"log/slog"
	"testing"

	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/auditing/memory"
	"github.com/stretchr/testify/require"
)

func TestAuditing_Memory(t *testing.T) {
	Run(t, Suite{
		Backend: func(t testing.TB) auditing.Auditing {
			a, err := memory.NewMemory(auditing.Config{Log: slog.Default()}, memory.MemoryConfig{})
			require.NoError(t, err)
			return a
		},
		// the memory backend matches body filters as contiguous substrings only
		Profile: Profile{Capabilities: []Capability{
			CapabilitySearch,
		}},
		HealthMessage: `audit backend "memory" is healthy`,
	})
}
