package util

import "strings"

// TrimSuffixes removes the first matching suffix from the input string.
func TrimSuffixes(s string, suffixes ...string) string {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return strings.TrimSuffix(s, suffix)
		}
	}
	return s
}
