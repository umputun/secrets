package store

import (
	"fmt"
	"math/rand"
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
	Data    string
	PinHash string `json:"-"`
	Errors  int    `json:"-"`
}

// Engine defines interface to save and load messages
type Engine interface {
	Save(msg *Message) (err error)
	Load(key string) (resutl *Message, err error)
	IncErr(key string) (count int, err error)
	Remove(key string) (err error)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
