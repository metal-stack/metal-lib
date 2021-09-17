package auth

import (
	"os"
	"path"
	"path/filepath"
)

const (
	RecommendedConfigPathEnvVar = "KUBECONFIG"
	RecommendedHomeDir          = ".kube"
	RecommendedFileName         = "config"
)

var (
	RecommendedConfigDir = path.Join(HomeDir(), RecommendedHomeDir)
	RecommendedHomeFile  = path.Join(RecommendedConfigDir, RecommendedFileName)
)

// returns the paths from env, may be empty or contain multiple paths
func fromEnv() []string {

	var paths []string

	envVarFiles := os.Getenv(RecommendedConfigPathEnvVar)
	if len(envVarFiles) != 0 {
		fileList := filepath.SplitList(envVarFiles)
		// prevent the same path load multiple times
		paths = deduplicate(fileList)
	}

	return paths
}

// deduplicate removes any duplicated values and returns a new slice, keeping the order unchanged
func deduplicate(s []string) []string {
	encountered := map[string]bool{}
	ret := make([]string, 0)
	for i := range s {
		if encountered[s[i]] {
			continue
		}
		encountered[s[i]] = true
		ret = append(ret, s[i])
	}
	return ret
}

// HomeDir returns the home directory for the current user
func HomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}
