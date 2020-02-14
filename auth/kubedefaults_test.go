package auth

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestFromEnvMultiplePath(t *testing.T) {

	os.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/another/path")
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	s := fromEnv()
	assert.Equal(t, "/tmp/path/to/kubeconfig", s[0])
	assert.Equal(t, "/another/path", s[1])
}

func TestFromEnvMultiplePathDeDup(t *testing.T) {

	os.Setenv(RecommendedConfigPathEnvVar, "/tmp/path/to/kubeconfig:/tmp/path/to/kubeconfig")
	defer os.Setenv(RecommendedConfigPathEnvVar, "")

	s := fromEnv()
	assert.Len(t, s, 1)
	assert.Equal(t, "/tmp/path/to/kubeconfig", s[0])
}

func TestFromEnvEmpty(t *testing.T) {

	os.Setenv(RecommendedConfigPathEnvVar, "")

	s := fromEnv()
	assert.Len(t, s, 0)
}
