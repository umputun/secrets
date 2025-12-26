package messager

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/nacl/secretbox"
)

// Crypt data with a global key + pin.
// It provides NaCl secretbox encryption (XSalsa20-Poly1305) for data
// needed to prevent storing it in naked form even in_memory storage.
type Crypt struct {
	Key string
}

// Request for both Encrypt and Decrypt
type Request struct {
	Pin  string
	Data []byte
}

// Encrypt encrypts data with secretbox and returns base64-encoded result
func (c Crypt) Encrypt(req Request) ([]byte, error) {
	keyWithPin := fmt.Sprintf("%s%s", c.Key, req.Pin)
	if len(keyWithPin) != 32 {
		return nil, fmt.Errorf("key+pin should be 32 bytes, got %d", len(keyWithPin))
	}

	naclKey := new([32]byte)
	copy(naclKey[:], keyWithPin[:32])
	nonce := new([24]byte)
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("could not read from random: %w", err)
	}
	out := make([]byte, 24)
	copy(out, nonce[:])
	sealed := secretbox.Seal(out, req.Data, nonce, naclKey)
	return []byte(base64.StdEncoding.EncodeToString(sealed)), nil
}

// Decrypt decrypts base64-encoded data with secretbox
func (c Crypt) Decrypt(req Request) ([]byte, error) {
	keyWithPin := fmt.Sprintf("%s%s", c.Key, req.Pin)
	if len(keyWithPin) != 32 {
		return nil, errors.New("key+pin should be 32 bytes")
	}

	naclKey := new([32]byte)
	copy(naclKey[:], keyWithPin[:32])

	sealed, err := base64.StdEncoding.DecodeString(string(req.Data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	nonce := new([24]byte)
	copy(nonce[:], sealed[:24])

	decrypted, ok := secretbox.Open(nil, sealed[24:], nonce, naclKey)
	if !ok {
		return nil, errors.New("failed to decrypt")
	}
	return decrypted, nil
}

// MakeSignKey creates 32-pin bytes signKey for AES256
func MakeSignKey(signKey string, pinSize int) (result string) {
	if len(signKey) >= 32-pinSize {
		return signKey[:32-pinSize]
	}

	repeatCount := (32 - pinSize) / len(signKey)
	result = strings.Repeat(signKey, repeatCount+1)

	return result[:32-pinSize]
}
