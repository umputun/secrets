// Package messager package using injected engine.Store to save and load messages.
// It does all encryption/decryption and hashing. Store used as a dump storage only.
// Passed (from user) pin used as a part of encryption key for data and delegated to crypt.Crypt.
// Pins not saved directly, but hashed with bcrypt.
package messager

import (
	"fmt"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/umputun/secrets/app/store"
)

//go:generate moq -out crypt_mock.go -fmt goimports . Crypter
//go:generate moq -out engine_mock.go -fmt goimports . Engine

// Errors
var (
	ErrBadPin        = fmt.Errorf("wrong pin")
	ErrBadPinAttempt = fmt.Errorf("wrong pin attempt")
	ErrCrypto        = fmt.Errorf("crypto error")
	ErrInternal      = fmt.Errorf("internal error")
	ErrExpired       = fmt.Errorf("message expired")
	ErrDuration      = fmt.Errorf("bad duration")
	ErrFileTooLarge  = fmt.Errorf("file too large")
	ErrBadFileName   = fmt.Errorf("invalid file name")
)

// filePrefix marks file messages with unencrypted metadata
// format: !!FILE!!filename!!content-type!!\n<encrypted binary>
const filePrefix = "!!FILE!!"

// MessageProc creates and save messages and retrieve per key
type MessageProc struct {
	Params
	crypt  Crypter
	engine Engine
}

// Params to customize limits
type Params struct {
	MaxDuration    time.Duration
	MaxPinAttempts int
	MaxFileSize    int64
}

// FileRequest contains data for file message creation
type FileRequest struct {
	Duration    time.Duration
	Pin         string
	FileName    string
	ContentType string
	Data        []byte
}

// Crypter interface wraps crypt methods
type Crypter interface {
	Encrypt(req Request) (result []byte, err error)
	Decrypt(req Request) (result []byte, err error)
}

// Engine defines interface to save, load, remove and inc errors count for messages
type Engine interface {
	Save(msg *store.Message) (err error)
	Load(key string) (result *store.Message, err error)
	IncErr(key string) (count int, err error)
	Remove(key string) (err error)
}

// New makes MessageProc with the engine and crypt
func New(engine Engine, crypter Crypter, params Params) *MessageProc {

	if params.MaxDuration == 0 {
		params.MaxDuration = time.Hour * 24 * 31 // 31 days if nothing defined
	}
	if params.MaxPinAttempts == 0 {
		params.MaxPinAttempts = 3
	}
	if params.MaxFileSize == 0 {
		params.MaxFileSize = 1024 * 1024 // 1MB default
	}
	log.Printf("[INFO] created messager with %+v", params)

	return &MessageProc{
		engine: engine,
		crypt:  crypter,
		Params: params,
	}
}

// MakeMessage from data, pin and duration, saves to engine. Encrypts data part with pin.
func (p MessageProc) MakeMessage(duration time.Duration, msg, pin string) (result *store.Message, err error) {

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

	key := uuid.New().String()

	exp := time.Now().Add(duration)
	result = &store.Message{
		Key:     store.Key(exp, key),
		Exp:     time.Now().Add(duration),
		PinHash: pinHash,
	}

	result.Data, err = p.crypt.Encrypt(Request{Data: []byte(msg), Pin: pin})
	if err != nil {
		log.Printf("[ERROR] failed to encrypt, %v", err)
		return nil, ErrCrypto
	}
	err = p.engine.Save(result)
	return result, err
}

// LoadMessage gets from engine, verifies Message with pin and decrypts content.
// It also removes accessed messages and invalidate them on multiple wrong pins.
// Message decrypted by this function will be returned naked to consumer.
func (p MessageProc) LoadMessage(key, pin string) (msg *store.Message, err error) {

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
		count, e := p.engine.IncErr(key)
		if e != nil {
			return nil, ErrBadPin
		}
		log.Printf("[WARN] wrong pin provided for %s (%d times)", key, count)
		if count >= p.MaxPinAttempts {
			_ = p.engine.Remove(key)
			return nil, ErrBadPin
		}
		return msg, ErrBadPinAttempt
	}

	// for file messages, everything after !!FILE!! is encrypted (including metadata)
	// for text messages, the entire data is encrypted
	dataToDecrypt := msg.Data
	isFile := IsFileMessage(msg.Data)
	if isFile {
		// skip !!FILE!! prefix, decrypt the rest (metadata + binary)
		dataToDecrypt = msg.Data[len(filePrefix):]
	}

	r, err := p.crypt.Decrypt(Request{Data: dataToDecrypt, Pin: pin})
	if err != nil {
		log.Printf("[WARN] can't decrypt, %v", err)
		_ = p.engine.Remove(key)
		return nil, ErrBadPin
	}

	if err := p.engine.Remove(key); err != nil {
		log.Printf("[WARN] failed to remove, %v", err)
	}

	// for file messages, prepend !!FILE!! to decrypted result for ParseFileHeader compatibility
	// decrypted result is: filename!!content-type!!\n<binary>
	if isFile {
		msg.Data = append([]byte(filePrefix), r...)
	} else {
		msg.Data = r
	}
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
		return "", fmt.Errorf("can't make hashed pin: %w", err)
	}
	return string(hashedPin), nil
}

// IsFile checks if a message with given key is a file message (without decrypting).
// Returns false if message doesn't exist or is not a file.
func (p MessageProc) IsFile(key string) bool {
	msg, err := p.engine.Load(key)
	if err != nil {
		return false
	}
	return IsFileMessage(msg.Data)
}

// MakeFileMessage creates a message from file data with unencrypted prefix for metadata.
// Format: !!FILE!!filename!!content-type!!\n<encrypted binary>
func (p MessageProc) MakeFileMessage(req FileRequest) (result *store.Message, err error) {
	if req.Pin == "" {
		log.Printf("[WARN] save rejected, empty pin")
		return nil, ErrBadPin
	}

	if req.FileName == "" || len(req.FileName) > 255 || strings.Contains(req.FileName, "!!") ||
		strings.ContainsAny(req.FileName, "\n\r") || strings.Contains(req.FileName, "..") ||
		strings.ContainsAny(req.FileName, "/\\") {
		log.Printf("[WARN] save rejected, invalid file name")
		return nil, ErrBadFileName
	}

	if int64(len(req.Data)) > p.MaxFileSize {
		log.Printf("[WARN] save rejected, file too large: %d > %d", len(req.Data), p.MaxFileSize)
		return nil, ErrFileTooLarge
	}

	if req.Duration > p.MaxDuration {
		log.Printf("[ERROR] can't use duration, %v > %v", req.Duration, p.MaxDuration)
		return nil, ErrDuration
	}

	pinHash, err := p.makeHash(req.Pin)
	if err != nil {
		log.Printf("[ERROR] can't hash pin, %v", err)
		return nil, ErrInternal
	}

	// build metadata header and combine with binary data for encryption
	// format after decryption: filename!!content-type!!\n<binary>
	metadata := fmt.Sprintf("%s!!%s!!\n", req.FileName, req.ContentType)
	dataToEncrypt := append([]byte(metadata), req.Data...)

	// encrypt metadata + binary together (only !!FILE!! prefix stays unencrypted)
	encryptedData, err := p.crypt.Encrypt(Request{Data: dataToEncrypt, Pin: req.Pin})
	if err != nil {
		log.Printf("[ERROR] failed to encrypt file, %v", err)
		return nil, ErrCrypto
	}

	// stored format: !!FILE!!<encrypted blob containing metadata+data>
	fullData := append([]byte(filePrefix), encryptedData...)

	key := uuid.New().String()
	exp := time.Now().Add(req.Duration)
	result = &store.Message{
		Key:     store.Key(exp, key),
		Exp:     exp,
		PinHash: pinHash,
		Data:    fullData,
	}

	err = p.engine.Save(result)
	return result, err
}

// IsFileMessage checks if message data has file prefix (works on raw stored data)
func IsFileMessage(data []byte) bool {
	return len(data) > len(filePrefix) && string(data[:len(filePrefix)]) == filePrefix
}

// ParseFileHeader extracts filename, content-type, and data start position from file message.
// Returns empty strings and -1 if not a valid file message.
func ParseFileHeader(data []byte) (filename, contentType string, dataStart int) {
	if !IsFileMessage(data) {
		return "", "", -1
	}

	// format: !!FILE!!filename!!content-type!!\n<data>
	// find newline which marks end of header
	headerEnd := -1
	for i := len(filePrefix); i < len(data); i++ {
		if data[i] == '\n' {
			headerEnd = i
			break
		}
	}
	if headerEnd == -1 {
		return "", "", -1
	}

	// parse header: filename!!content-type!!
	header := string(data[len(filePrefix):headerEnd])
	parts := strings.Split(header, "!!")
	if len(parts) < 2 {
		return "", "", -1
	}

	return parts[0], parts[1], headerEnd + 1
}
