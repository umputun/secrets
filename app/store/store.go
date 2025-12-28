// Package store defines and implements data store for sqlite and in-memory
package store

import (
	"crypto/rand"
	"errors"
	"time"
)

// Error messages
var (
	ErrLoadRejected = errors.New("message expired or deleted")
	ErrSaveRejected = errors.New("can't save message")
)

// Message with key and exp. time
type Message struct {
	Key       string
	Exp       time.Time
	Data      []byte
	PinHash   string
	Errors    int
	ClientEnc bool // true if client-side encrypted (UI), false if server-side (API)
}

// base62 alphabet for short ID generation
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// GenerateID creates a 12-character base62 random ID using crypto/rand.
// Uses rejection sampling to avoid modulo bias.
func GenerateID() string {
	result := make([]byte, 12)
	for i := 0; i < 12; {
		b := make([]byte, 1)
		if _, err := rand.Read(b); err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		// reject values >= 248 to avoid modulo bias (248 = 62*4, evenly divisible)
		if b[0] < 248 {
			result[i] = alphabet[b[0]%62] //nolint:gosec // index always 0-61, safe
			i++
		}
	}
	return string(result)
}
