package main

import (
	"strings"
)

// ParseTag parses a struct tag string like `json:"name" db:"col"` into key→value map.
func ParseTag(raw string) map[string]string {
	result := make(map[string]string)
	raw = strings.TrimSpace(raw)

	for raw != "" {
		// Skip whitespace
		raw = strings.TrimLeft(raw, " \t")
		if raw == "" {
			break
		}

		// Find key (up to colon)
		colonIdx := strings.IndexByte(raw, ':')
		if colonIdx < 0 {
			break
		}
		key := raw[:colonIdx]
		raw = raw[colonIdx+1:]

		// Value must start with quote
		if len(raw) == 0 || raw[0] != '"' {
			break
		}
		raw = raw[1:]

		// Find closing quote
		quoteIdx := strings.IndexByte(raw, '"')
		if quoteIdx < 0 {
			// Unterminated — take the rest
			result[key] = raw
			break
		}
		result[key] = raw[:quoteIdx]
		raw = raw[quoteIdx+1:]
	}

	return result
}

// SetTagKey sets or updates a single tag key, preserving other keys and their order.
func SetTagKey(raw, key, value string) string {
	tags := parseTagOrdered(raw)

	found := false
	for i, t := range tags {
		if t.key == key {
			tags[i].value = value
			found = true
			break
		}
	}
	if !found {
		tags = append(tags, tagPair{key: key, value: value})
	}

	return buildTag(tags)
}

// RemoveTagKey removes a single tag key. If key is empty, removes the entire tag.
func RemoveTagKey(raw, key string) string {
	if key == "" {
		return ""
	}
	tags := parseTagOrdered(raw)
	var filtered []tagPair
	for _, t := range tags {
		if t.key != key {
			filtered = append(filtered, t)
		}
	}
	return buildTag(filtered)
}

type tagPair struct {
	key   string
	value string
}

func parseTagOrdered(raw string) []tagPair {
	var pairs []tagPair
	raw = strings.TrimSpace(raw)

	for raw != "" {
		raw = strings.TrimLeft(raw, " \t")
		if raw == "" {
			break
		}

		colonIdx := strings.IndexByte(raw, ':')
		if colonIdx < 0 {
			break
		}
		key := raw[:colonIdx]
		raw = raw[colonIdx+1:]

		if len(raw) == 0 || raw[0] != '"' {
			break
		}
		raw = raw[1:]

		quoteIdx := strings.IndexByte(raw, '"')
		if quoteIdx < 0 {
			pairs = append(pairs, tagPair{key: key, value: raw})
			break
		}
		pairs = append(pairs, tagPair{key: key, value: raw[:quoteIdx]})
		raw = raw[quoteIdx+1:]
	}

	return pairs
}

func buildTag(tags []tagPair) string {
	if len(tags) == 0 {
		return ""
	}
	var parts []string
	for _, t := range tags {
		parts = append(parts, t.key+`:`+`"`+t.value+`"`)
	}
	return strings.Join(parts, " ")
}
