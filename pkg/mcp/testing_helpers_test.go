package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// jsonEqual compares two byte slices containing JSON, ignoring whitespace differences.
// Useful for comparing marshaled JSON in tests.
func jsonEqual(a, b []byte) (bool, error) {
	var j1, j2 interface{}
	if err := json.Unmarshal(a, &j1); err != nil {
		// Handle null input specifically for comparison
		if string(a) == "null" {
			j1 = nil
		} else {
			return false, fmt.Errorf("failed to unmarshal first JSON (%s): %w", string(a), err)
		}
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		// Handle null input specifically for comparison
		if string(b) == "null" {
			j2 = nil
		} else {
			return false, fmt.Errorf("failed to unmarshal second JSON (%s): %w", string(b), err)
		}
	}
	return reflect.DeepEqual(j1, j2), nil
}
