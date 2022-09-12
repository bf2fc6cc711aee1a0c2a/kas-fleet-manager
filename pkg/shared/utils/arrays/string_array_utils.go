package arrays

import "fmt"

// FindFirstString Finds the first element into the passed in values where the predicate is true
// Returns: -1 and empty string when not found, position and the string itself otherwise.
var FindFirstString = FindFirst[string]
var FilterStringSlice = Filter[string]

// FirstNonEmpty Returns the first non-empty string between the passed in values.
// If no non-empty string can be found, returns an empty string and an error
func FirstNonEmpty(values ...string) (string, error) {
	if idx, _ := FindFirst(values, func(s string) bool { return s != "" }); idx == -1 {
		return "", fmt.Errorf("all strings are empty")
	} else {
		return values[idx], nil
	}
}

// FirstNonEmptyOrDefault Returns the first non-empty string between the passed in values.
// If no non-empty string can be found, returns defaultValue
func FirstNonEmptyOrDefault(defaultValue string, values ...string) string {
	if idx, _ := FindFirst(values, func(s string) bool { return s != "" }); idx == -1 {
		return defaultValue
	} else {
		return values[idx]
	}
}
