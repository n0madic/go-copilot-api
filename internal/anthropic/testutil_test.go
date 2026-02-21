package anthropic_test

import "strings"

func strPtr(v string) *string {
	return &v
}

func tempPtr(v float64) *float64 {
	return &v
}

func containsAll(haystack string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}
