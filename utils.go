package monarch

import (
	"strings"
	"unicode"
)

func toSnakeCase(s string) string {
	var result []rune
	var prev rune

	for i, current := range s {
		// Handle non-alphanumeric characters
		if !unicode.IsLetter(current) && !unicode.IsDigit(current) {
			if len(result) > 0 && result[len(result)-1] != '_' {
				result = append(result, '_')
			}
			prev = current
			continue
		}

		// Handle alphanumeric characters
		currentLower := unicode.ToLower(current)
		if i > 0 {
			prevIsLower := unicode.IsLower(prev)
			prevIsUpper := unicode.IsUpper(prev)
			prevIsNumber := unicode.IsDigit(prev)

			currentIsUpper := unicode.IsUpper(current)

			// Add underscore between:
			// 1. Lowercase/Number and Uppercase
			// 2. Uppercase and Uppercase followed by Lowercase
			if (prevIsLower || prevIsNumber) && currentIsUpper ||
				(prevIsUpper && currentIsUpper && i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
				if len(result) > 0 && result[len(result)-1] != '_' {
					result = append(result, '_')
				}
			}
		}

		result = append(result, currentLower)
		prev = current
	}

	// Remove consecutive underscores and trim
	cleaned := []rune{}
	for _, r := range result {
		if r == '_' {
			if len(cleaned) == 0 || cleaned[len(cleaned)-1] != '_' {
				cleaned = append(cleaned, r)
			}
		} else {
			cleaned = append(cleaned, r)
		}
	}

	return strings.Trim(string(cleaned), "_")
}
