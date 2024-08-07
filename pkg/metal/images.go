package metal

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// GetOsAndSemverFromImage parses a metal-api image ID to OS and Semver or returns an error
//
// The last part must be the semantic version, valid ids are:
//
// ubuntu-19.04                 => os: ubuntu         version: 19.04
// ubuntu-19.04.20200408        => os: ubuntu         version: 19.04.20200408
// ubuntu-small-19.04.20200408  => os: ubuntu-small   version: 19.04.20200408
func GetOsAndSemverFromImage(id string) (string, *semver.Version, error) {
	imageParts := strings.Split(id, "-")
	if len(imageParts) < 2 {
		return "", nil, fmt.Errorf("invalid format for os image, expected <os>-<major>.<minor>[.<patch>]: %s", id)
	}

	var (
		parts   = len(imageParts) - 1
		os      = strings.Join(imageParts[:parts], "-")
		version = strings.Join(imageParts[parts:], "")
	)

	v, err := semver.NewVersion(version)
	if err != nil {
		return "", nil, err
	}

	return os, v, nil
}
