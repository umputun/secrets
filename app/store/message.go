package store

import (
	"fmt"
	"log"
	"time"

	"github.com/nu7hatch/gouuid"
	"github.com/umputun/secrets/app/crypt"

	"golang.org/x/crypto/bcrypt"
)

// Errors
var (
	ErrBadPin        = fmt.Errorf("wrong pin")
	ErrBadPinAttempt = fmt.Errorf("wrong pin attempt")
	ErrCrypto        = fmt.Errorf("crypto errpr")
	ErrInternal      = fmt.Errorf("internal error")
	ErrExpired       = fmt.Errorf("message expired")
)

// MessageProc acts as a factory for new messages and decoder for existing messages
type messageProc struct {
	crypt.Crypt
	maxDuration time.Duration
}

// Message with key and exp. time
type Message struct {
	Key     string
	Exp     time.Time
	Data    string
	PinHash string `json:"-"`
	Errors  int    `json:"-"`
}

// MakeMessage from data, ping and duration. Encrypt data part with pin
func (p messageProc) MakeMessage(duration time.Duration, msg string, pin string) (result *Message, err error) {

	if pin == "" {
		log.Printf("[WARN] save rejected, empty pin")
		return nil, ErrBadPin
	}

	pinHash, err := p.MakeHash(pin)
	if err != nil {
		log.Printf("[ERROR] can't hash pin, %v", err)
		return nil, ErrInternal
	}

	key, err := uuid.NewV4()
	if err != nil {
		log.Printf("[ERROR] can't make uuis, %v", err)
		return nil, ErrInternal
	}

	result = &Message{
		Key:     key.String(),
		Exp:     time.Now().Add(duration),
		PinHash: pinHash,
	}

	result.Data, err = p.Encrypt(crypt.Request{Data: msg, Pin: pin})
	if err != nil {
		log.Printf("[ERROR] failed to encrypt, %v", err)
		return nil, ErrCrypto
	}

	return result, nil
}

// FromMessage verifies Message with pin and Decrypt content
func (p messageProc) FromMessage(msg Message, pin string) (result *Message, err error) {

	result = &msg
	if time.Now().After(msg.Exp) {
		log.Printf("[WARN] expired %s on %v", msg.Key, msg.Exp)
		return nil, ErrExpired
	}

	if !p.CheckHash(msg, pin) {
		result.Errors++
		log.Printf("[WARN] wrong pin provided (%d times)", msg.Errors)
		if result.Errors > 3 {
			return nil, ErrBadPin
		}
		return result, ErrBadPinAttempt
	}

	r, err := p.Decrypt(crypt.Request{Data: msg.Data, Pin: pin})
	if err != nil {
		log.Printf("[WARN] can't decrypt, %v", err)
		return nil, ErrBadPin

	}
	result.Data = r
	return result, nil

}

// CheckHash verifies msg.PinHash with provided pin
func (p messageProc) CheckHash(msg Message, pin string) bool {
	return bcrypt.CompareHashAndPassword([]byte(msg.PinHash), []byte(pin)) == nil
}

// MakeHash from pin
func (p messageProc) MakeHash(pin string) (result string, err error) {
	hashedPin, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return result, err
	}
	return string(hashedPin), nil
}
