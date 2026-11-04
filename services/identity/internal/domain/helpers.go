package domain

import (
	"strings"
	"time"
	"unicode"
)

func MinimumTime(first, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}

func ClonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}

	cloned := *value
	return &cloned
}

func NormalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}

	return &normalized
}

func NormalizeRequiredText(value string) (string, bool) {
	value = strings.TrimSpace(value)
	return value, value != ""
}

func HasText(value string) bool {
	return strings.TrimSpace(value) != ""
}

func NormalizeToken(value string) (string, bool) {
	value, ok := NormalizeRequiredText(value)
	return value, ok && strings.IndexFunc(value, unicode.IsSpace) < 0
}

func EqualOptionalText(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func HasDuplicates[T comparable](values []T) bool {
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}

	return false
}
