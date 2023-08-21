package tag

import (
	"fmt"
	"strings"
)

type TagMap map[string]string

// New returns a new tag in the format key=value
func New(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

// NewTagMap constructs a map of a list of labels.
func NewTagMap(labels []string) TagMap {
	result := TagMap{}
	for _, l := range labels {
		key, value, _ := strings.Cut(l, "=")
		result[key] = value
	}
	return result
}

// Contains returns true when the given key is contained in the label map.
func (tm TagMap) Contains(key, value string) bool {
	v, ok := tm[key]
	if !ok {
		return false
	}
	return v == value
}

// Value returns true when the label map contains the given key and returns the corresponding value.
func (tm TagMap) Value(key string) (string, bool) {
	value, ok := tm[key]
	return value, ok
}

// Get returns the whole tag when the given key is contained in the label map.
func (tm TagMap) Get(key string) (string, error) {
	value, ok := tm.Value(key)
	if !ok {
		return "", fmt.Errorf("no tag with key %q found", key)
	}

	return New(key, value), nil
}
