// Messager package using injected store.Engine to save and load messages.
// It does all encryption/decription and hashing. Engine used as a dump storage only.
// Passed (from user) pin used as a part of encryption key for data and delegated to crypt.Crypt.
// Pin is not saved directly, but hashed with bcrypt.

package messager

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
	ErrDuration      = fmt.Errorf("bad duration")
)

// MessageProc creates and save messages and retrive per key
type MessageProc struct {
	crypt.Crypt
	Params
	engine store.Engine
}

// Params to customzie limits
type Params struct {
	MaxDuration    time.Duration
	MaxPinAttempts int
}

// New makes MessageProc with engine and crypt
func New(engine store.Engine, crypt crypt.Crypt, params Params) *MessageProc {

	if params.MaxDuration == 0 {
		params.MaxDuration = time.Hour * 24 * 31 //31 days if nothing defined
	}
	if params.MaxPinAttempts == 0 {
		params.MaxPinAttempts = 3
	}
	log.Printf("[INFO] created messager with %+v", params)

	return &MessageProc{
		engine: engine,
		Crypt:  crypt,
		Params: params,
	}
}

// MakeMessage from data, pin and duration, saves to store. Encrypts data part with pin.
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

	if duration > p.MaxDuration {
		log.Printf("[ERROR] can't use duration, %v > %v", duration, p.MaxDuration)
		return nil, ErrDuration
	}

	key, err := uuid.NewV4()
	if err != nil {
		log.Printf("[ERROR] can't make uuid, %v", err)
		return nil, ErrInternal
	}

	exp := time.Now().Add(duration)
	result = &store.Message{
		Key:     fmt.Sprintf("%d-%s", exp.Unix(), key.String()),
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

// LoadMessage gets from store, verifies Message with pin and decrypts content.
// It also removes accessed messages and invalidate them on multiple wrong pins.
// Message decrypted by this function will be returned naked to consumer.
func (p MessageProc) LoadMessage(key string, pin string) (msg *store.Message, err error) {

	msg, err = p.engine.Load(key)
	if err != nil {
		return nil, err
	}

	if time.Now().After(msg.Exp) {
		log.Printf("[WARN] expired %s on %v", msg.Key, msg.Exp)
		_ = p.engine.Remove(key)
		return nil, ErrExpired
	}

	if !p.checkHash(msg, pin) {
		_, _ = p.engine.IncErr(key)
		log.Printf("[WARN] wrong pin provided (%d times)", msg.Errors+1)
		if msg.Errors > p.MaxPinAttempts {
			_ = p.engine.Remove(key)
			return nil, ErrBadPin
		}
		return msg, ErrBadPinAttempt
	}

	r, err := p.Decrypt(crypt.Request{Data: msg.Data, Pin: pin})
	if err != nil {
		log.Printf("[WARN] can't decrypt, %v", err)
		_ = p.engine.Remove(key)
		return nil, ErrBadPin

	}

	if err := p.engine.Remove(key); err != nil {
		log.Printf("[WARN] failed to remove, %v", err)
	}
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
		return "", err
	}
	return string(hashedPin), nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
