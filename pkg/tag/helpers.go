package tag

import "strings"

type TagMap map[string]string

// NewLabelMap constructs a map of a list of labels.
func NewTagMap(labels []string) TagMap {
	result := TagMap{}
	for _, l := range labels {
		key, value, _ := strings.Cut(l, "=")
		result[key] = value
	}
	return result
}

// Contains returns true when the given tag is contained in the label map.
func (tm TagMap) Contains(tag, value string) bool {
	v, ok := tm[tag]
	if !ok {
		return false
	}
	return v == value
}

// Value returns true when the label map contains the given tag and returns the corresponding value.
func (tm TagMap) Value(tag string) (string, bool) {
	value, ok := tm[tag]
	return value, ok
}
