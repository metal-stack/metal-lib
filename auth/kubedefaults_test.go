package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromEnvMultiplePath(t *testing.T) {
	t.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/another/path")
	s := fromEnv()
	assert.Equal(t, "/tmp/path/to/kubeconfig", s[0])
	assert.Equal(t, "/another/path", s[1])
}

func TestFromEnvMultiplePathDeDup(t *testing.T) {
	t.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/tmp/path/to/kubeconfig")
	s := fromEnv()
	assert.Len(t, s, 1)
	assert.Equal(t, "/tmp/path/to/kubeconfig", s[0])
}

func TestFromEnvEmpty(t *testing.T) {
	t.Setenv(RecommendedConfigPathEnvVar, "")
	s := fromEnv()
	assert.Empty(t, s)
}
