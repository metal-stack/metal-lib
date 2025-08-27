package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromEnvMultiplePath(t *testing.T) {
	err := os.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/another/path")
	require.NoError(t, err)
	defer func() {
		err := os.Setenv(RecommendedConfigPathEnvVar, "")
		assert.NoError(t, err)
	}()

	s := fromEnv()
	assert.Equal(t, "/tmp/path/to/kubeconfig", s[0])
	assert.Equal(t, "/another/path", s[1])
}

func TestFromEnvMultiplePathDeDup(t *testing.T) {
	err := os.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/tmp/path/to/kubeconfig")
	require.NoError(t, err)
	defer func() {
		err := os.Setenv(RecommendedConfigPathEnvVar, "")
		assert.NoError(t, err)
	}()

	s := fromEnv()
	assert.Len(t, s, 1)
	assert.Equal(t, "/tmp/path/to/kubeconfig", s[0])
}

func TestFromEnvEmpty(t *testing.T) {
	err := os.Setenv(RecommendedConfigPathEnvVar, "")
	require.NoError(t, err)

	s := fromEnv()
	assert.Empty(t, s)
}
