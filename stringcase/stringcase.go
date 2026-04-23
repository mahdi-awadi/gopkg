// Package stringcase converts between common case styles: camelCase,
// PascalCase, snake_case, SCREAMING_SNAKE_CASE, and kebab-case.
//
// The input is first split into words (via rune classification + boundaries),
// then re-joined in the target style. Handles ASCII only — for Unicode
// segmentation use a dedicated library.
//
// Zero third-party deps.
package stringcase

import (
	"strings"
	"unicode"
)

// Split splits s into words by case transitions, digits boundaries, and
// non-alphanumeric separators.
func Split(s string) []string {
	var words []string
	var cur []rune
	prev := rune(0)
	for _, r := range s {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.':
			if len(cur) > 0 {
				words = append(words, string(cur))
				cur = cur[:0]
			}
		case unicode.IsUpper(r):
			if len(cur) > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
				words = append(words, string(cur))
				cur = cur[:0]
			}
			cur = append(cur, r)
		case unicode.IsLower(r):
			if len(cur) > 1 && unicode.IsUpper(prev) && len(cur) >= 2 && unicode.IsUpper(cur[len(cur)-2]) {
				// Handle acronym boundary like "HTTPRequest" → "HTTP","Request":
				// when the previous char was upper and the one before it was too,
				// the previous char actually starts the next word.
				next := cur[len(cur)-1]
				cur = cur[:len(cur)-1]
				words = append(words, string(cur))
				cur = []rune{next}
			}
			cur = append(cur, r)
		case unicode.IsDigit(r):
			cur = append(cur, r)
		default:
			// drop other runes
		}
		prev = r
	}
	if len(cur) > 0 {
		words = append(words, string(cur))
	}
	return words
}

// Snake returns snake_case.
func Snake(s string) string {
	words := Split(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

// ScreamingSnake returns UPPER_SNAKE_CASE.
func ScreamingSnake(s string) string {
	words := Split(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w)
	}
	return strings.Join(words, "_")
}

// Kebab returns kebab-case.
func Kebab(s string) string {
	words := Split(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "-")
}

// Camel returns camelCase (first word lowercase, subsequent words capitalized).
func Camel(s string) string {
	words := Split(s)
	var b strings.Builder
	for i, w := range words {
		if i == 0 {
			b.WriteString(strings.ToLower(w))
		} else {
			b.WriteString(title(w))
		}
	}
	return b.String()
}

// Pascal returns PascalCase (every word capitalized).
func Pascal(s string) string {
	words := Split(s)
	var b strings.Builder
	for _, w := range words {
		b.WriteString(title(w))
	}
	return b.String()
}

// title capitalizes the first rune and lowercases the rest. We avoid
// strings.Title (deprecated) and golang.org/x/text/cases (third-party).
func title(w string) string {
	if w == "" {
		return w
	}
	rs := []rune(strings.ToLower(w))
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}
