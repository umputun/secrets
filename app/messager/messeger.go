package messager

// Messager package using injected store.Engine to save and load messages.
// It does all enccytption/decription, hasing and uses engine as dump storage only.
// Passed (from user) pin used as a part of encryption key for data and get delegated to crypt.Crypt.
// Pin not saved directly, but hashed with bcrypt.
import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/nu7hatch/gouuid"
	"github.com/umputun/secrets/app/crypt"
	"github.com/umputun/secrets/app/store"
)

// Errors
var (
	ErrBadPin        = fmt.Errorf("wrong pin")
	ErrBadPinAttempt = fmt.Errorf("wrong pin attempt")
	ErrCrypto        = fmt.Errorf("crypto errpr")
	ErrInternal      = fmt.Errorf("internal error")
	ErrExpired       = fmt.Errorf("message expired")
)

// MessageProc creates and save messages and retrive per key
type MessageProc struct {
	crypt.Crypt
	maxDuration time.Duration
	engine      store.Engine
}

// New makes MessageProc with engine and crypt
func New(engine store.Engine, crypt crypt.Crypt, maxDuration time.Duration) *MessageProc {
	return &MessageProc{engine: engine, Crypt: crypt, maxDuration: maxDuration}
}

// MakeMessage from data, ping and duration. Encrypt data part with pin
func (p MessageProc) MakeMessage(duration time.Duration, msg string, pin string) (result *store.Message, err error) {

	if pin == "" {
		log.Printf("[WARN] save rejected, empty pin")
		return nil, ErrBadPin
	}

	pinHash, err := p.makeHash(pin)
	if err != nil {
		log.Printf("[ERROR] can't hash pin, %v", err)
		return nil, ErrInternal
	}

	key, err := uuid.NewV4()
	if err != nil {
		log.Printf("[ERROR] can't make uuis, %v", err)
		return nil, ErrInternal
	}

	result = &store.Message{
		Key:     key.String(),
		Exp:     time.Now().Add(duration),
		PinHash: pinHash,
	}

	result.Data, err = p.Encrypt(crypt.Request{Data: msg, Pin: pin})
	if err != nil {
		log.Printf("[ERROR] failed to encrypt, %v", err)
		return nil, ErrCrypto
	}
	err = p.engine.Save(result)
	return result, err
}

// LoadMessage get from store, verifies Message with pin and Decrypt content.
// It also removes accessed messages and invalidate on multiple wrong pins.
// Message decrypted by this function and returned naked to consumer.
func (p MessageProc) LoadMessage(key string, pin string) (msg *store.Message, err error) {

	msg, err = p.engine.Load(key)
	if err != nil {
		return nil, err
	}

	if time.Now().After(msg.Exp) {
		log.Printf("[WARN] expired %s on %v", msg.Key, msg.Exp)
		p.engine.Remove(key)
		return nil, ErrExpired
	}

	if !p.checkHash(msg, pin) {
		p.engine.IncErr(key)
		log.Printf("[WARN] wrong pin provided (%d times)", msg.Errors+1)
		if msg.Errors > 3 {
			p.engine.Remove(key)
			return nil, ErrBadPin
		}
		return msg, ErrBadPinAttempt
	}

	r, err := p.Decrypt(crypt.Request{Data: msg.Data, Pin: pin})
	if err != nil {
		log.Printf("[WARN] can't decrypt, %v", err)
		p.engine.Remove(key)
		return nil, ErrBadPin

	}
	p.engine.Remove(key)
	msg.Data = r
	return msg, nil

}

// checkHash verifies msg.PinHash with provided pin
func (p MessageProc) checkHash(msg *store.Message, pin string) bool {
	return bcrypt.CompareHashAndPassword([]byte(msg.PinHash), []byte(pin)) == nil
}

// makeHash from pin
func (p MessageProc) makeHash(pin string) (result string, err error) {
	hashedPin, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return result, err
	}
	return string(hashedPin), nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
