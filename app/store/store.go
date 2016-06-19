package store

import (
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Error messages
var (
	ErrNoSuchThing  = fmt.Errorf("message expired or deleted")
	ErrSaveRejected = fmt.Errorf("can't save message")
	ErrBadPin       = fmt.Errorf("wrong pin")
)

// Interface to save and load messages
type Interface interface {
	Save(duration time.Duration, msg string, pin string) (Message, error)
	Load(key string, pin string) (Message, error)
}

// Message with key and exp. time
type Message struct {
	Key     string
	Exp     time.Time
	Data    string
	PinHash string `json:"-"`
	Errors  int    `json:"-"`
}

// CheckHash verifies u.Hash with provided pin
func CheckHash(m Message, pin string) bool {
	return bcrypt.CompareHashAndPassword([]byte(m.PinHash), []byte(pin)) == nil
}

// MakeHash from password (cred)
func MakeHash(pin string) (result string, err error) {
	hashedPin, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return result, err
	}
	return string(hashedPin), nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
