package store

import (
	"fmt"
	"math/rand"
	"time"
)

// Error messages
var (
	ErrGetRejected  = fmt.Errorf("message expired or deleted")
	ErrSaveRejected = fmt.Errorf("can't save message")
)

// Interface to save and load messages
type Interface interface {
	Save(duration time.Duration, msg string, pin string) (result *Message, err error)
	Load(key string, pin string) (resutl *Message, err error)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
