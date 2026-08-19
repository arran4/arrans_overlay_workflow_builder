package util

import "strings"

// TrimSuffixes removes the first matching suffix from the input string.
func TrimSuffixes(s string, suffixes ...string) string {
	for _, suffix := range suffixes {
		if before, ok := strings.CutSuffix(s, suffix); ok {
			return before
		}
	}
	return s
}

func StringOrDefault(description *string, defaultStr string) string {
	if description == nil {
		return defaultStr
	}
	return *description
}
