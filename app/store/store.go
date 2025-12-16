// Package store defines and implements data store for boltdb and in-memory
package store

import (
	"fmt"
	"time"
)

// Error messages
var (
	ErrLoadRejected = fmt.Errorf("message expired or deleted")
	ErrSaveRejected = fmt.Errorf("can't save message")
)

// Message with key and exp. time
type Message struct {
	Key     string
	Exp     time.Time
	Data    []byte
	PinHash string
	Errors  int
	// file support fields
	IsFile      bool
	FileName    string
	ContentType string
	FileSize    int64
}

// Key makes store key with ts prefix
func Key(ts time.Time, key string) string {
	return fmt.Sprintf("%x-%s", ts.Unix(), key)
}
