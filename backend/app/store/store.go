// Package store defines and implements data store for boltdb and in-memory
package store

import (
	"fmt"
	"time"
)

//go:generate mockery -inpkg -name Engine -case snake

// Error messages
var (
	ErrLoadRejected = fmt.Errorf("message expired or deleted")
	ErrSaveRejected = fmt.Errorf("can't save message")
)

// Message with key and exp. time
type Message struct {
	Key     string
	Exp     time.Time
	Data    string
	PinHash string
	Errors  int
}

// Engine defines interface to save, load, remove and inc errors count for messages
type Engine interface {
	Save(msg *Message) (err error)
	Load(key string) (result *Message, err error)
	IncErr(key string) (count int, err error)
	Remove(key string) (err error)
}
