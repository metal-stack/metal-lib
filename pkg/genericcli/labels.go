package genericcli

import (
	"fmt"
	"strings"
)

// LabelsToMap splits strings at = and returns a corresponding map, errors out when there is no =.
func LabelsToMap(labels []string) (map[string]string, error) {
	labelMap := make(map[string]string)
	for _, l := range labels {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("provided labels must be in the form <key>=<value>, found: %s", l)
		}
		labelMap[parts[0]] = parts[1]
	}
	return labelMap, nil
}
