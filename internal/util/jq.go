package util

import (
	"fmt"
	"strconv"
	"strings"
)

// Jq is a helper function to extract a value from a JSON-like map using a path
func Jq(input map[string]interface{}, path string) (interface{}, error) { // Changed return type to interface{}
	path = strings.TrimPrefix(path, ".")
	keys := strings.Split(path, ".")
	var current interface{} = input

	for _, key := range keys {
		if currentMap, ok := current.(map[string]interface{}); ok {
			if strings.Contains(key, "[") && strings.Contains(key, "]") {
				arrayKey := key[:strings.Index(key, "[")]
				indexStr := key[strings.Index(key, "[")+1 : strings.Index(key, "]")]
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					return nil, fmt.Errorf("invalid array index in path: %s", key)
				}
				if array, ok := currentMap[arrayKey].([]interface{}); ok {
					if index < 0 || index >= len(array) {
						return nil, fmt.Errorf("index out of range in path: %s", key)
					}
					current = array[index]
				} else {
					return nil, fmt.Errorf("expected array at path: %s", key)
				}
			} else {
				current = currentMap[key]
			}
		} else {
			return nil, fmt.Errorf("expected map at path: %s", key)
		}
	}

	return current, nil // Return the final value without type assertion
}
