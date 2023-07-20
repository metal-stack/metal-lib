package k8s

import "github.com/Masterminds/semver/v3"

var (
	KubernetesV119 = semver.MustParse("1.19")
	KubernetesV120 = semver.MustParse("1.20")
	KubernetesV121 = semver.MustParse("1.21")
	KubernetesV122 = semver.MustParse("1.22")
	KubernetesV123 = semver.MustParse("1.23")
	KubernetesV124 = semver.MustParse("1.24")
	KubernetesV125 = semver.MustParse("1.25")
	KubernetesV126 = semver.MustParse("1.26")
	KubernetesV127 = semver.MustParse("1.27")
	KubernetesV128 = semver.MustParse("1.28")
)

func LessThan(actual string, target *semver.Version) (bool, error) {
	v, err := semver.NewVersion(actual)
	if err != nil {
		return false, err
	}

	return v.LessThan(target), nil
}
func GreaterThanOrEqual(actual string, target *semver.Version) (bool, error) {
	l, err := LessThan(actual, target)
	if err != nil {
		return false, err
	}
	return !l, nil
}
