package genericcli

import (
	"fmt"
	"strings"
)

// LabelsToMap splits strings at = and returns a corresponding map, errors out when there is no =.
func LabelsToMap(labels []string) (map[string]string, error) {
	labelMap := make(map[string]string)
	for _, l := range labels {
		key, value, found := strings.Cut(l, "=")
		if !found {
			return nil, fmt.Errorf("provided labels must be in the form <key>=<value>, found: %s", l)
		}
		labelMap[key] = value
	}
	return labelMap, nil
}

// MapToLabels returns a list of labels from a label map.
func MapToLabels(m map[string]string) []string {
	var res []string
	for k, v := range m {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return res
}
