// Package test provides the auditing conformance suite: a shared set of
// behavioural cases that every search-capable auditing backend must pass.
//
// A backend runs the suite through Run with a Suite describing how to build
// the backend under test and a Profile declaring its capabilities. Cases
// requiring capabilities the profile lacks are skipped with a visible
// reason, so parity gaps between backends stay declared instead of hiding
// in name-based skip lists.
package test

import (
	"fmt"
	"testing"

	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/pkg/healthstatus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Capability names a backend feature that conformance cases may require.
type Capability string

const (
	// CapabilitySearch indicates the backend implements Search. All current
	// conformance cases require it.
	CapabilitySearch Capability = "search"

	// CapabilityMultiWordBody indicates Body filters match entries containing
	// all filtered words, not only the contiguous phrase.
	CapabilityMultiWordBody Capability = "multi-word-body"
)

// Profile declares the capabilities of a backend under test.
type Profile struct {
	Capabilities []Capability
}

// Has reports whether the profile includes the given capability.
func (p Profile) Has(c Capability) bool {
	for _, pc := range p.Capabilities {
		if pc == c {
			return true
		}
	}
	return false
}

// Suite configures one run of the conformance cases.
type Suite struct {
	// Backend provides the system under test. It is called before every case
	// and for the final health check: stateful backends return a fresh
	// instance, shared backends return their long-lived instance.
	Backend func(t testing.TB) auditing.Auditing
	// Profile declares the backend's capabilities.
	Profile Profile
	// Cleanup resets shared backend state before each case. Optional.
	Cleanup func()
	// HealthMessage, if non-empty, is asserted verbatim as the message of the
	// backend's health check after all cases ran.
	HealthMessage string
}

// Run executes the conformance cases against the suite's backend and asserts
// its health check afterwards.
func Run(t *testing.T, s Suite) {
	t.Helper()

	if !s.Profile.Has(CapabilitySearch) {
		t.Fatalf("backend profile must support %q to run the conformance suite", CapabilitySearch)
	}

	for i, tc := range cases() {
		tc := tc
		t.Run(fmt.Sprintf("%d %s", i, tc.name), func(t *testing.T) {
			if missing := tc.missing(s.Profile); len(missing) > 0 {
				t.Skipf("backend does not support: %v", missing)
				return
			}
			if s.Cleanup != nil {
				s.Cleanup()
			}
			tc.t(t, s.Backend(t))
		})
	}

	healthResult, err := s.Backend(t).Check(t.Context())
	require.NoError(t, err)
	assert.Equal(t, healthstatus.HealthStatusHealthy, healthResult.Status)
	if s.HealthMessage != "" {
		assert.Equal(t, s.HealthMessage, healthResult.Message)
	}
}

// missing returns the capabilities the case needs that the profile lacks.
func (c conformanceCase) missing(p Profile) []Capability {
	var missing []Capability
	for _, cap := range c.caps {
		if !p.Has(cap) {
			missing = append(missing, cap)
		}
	}
	return missing
}
