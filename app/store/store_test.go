package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateID_Length(t *testing.T) {
	id := GenerateID()
	assert.Len(t, id, 12, "id should be 12 characters")
}

func TestGenerateID_Charset(t *testing.T) {
	id := GenerateID()
	for _, c := range id {
		isDigit := c >= '0' && c <= '9'
		isUpper := c >= 'A' && c <= 'Z'
		isLower := c >= 'a' && c <= 'z'
		assert.True(t, isDigit || isUpper || isLower, "character %c should be base62 (0-9, A-Z, a-z)", c)
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		id := GenerateID()
		require.False(t, seen[id], "duplicate id found: %s", id)
		seen[id] = true
	}
}
