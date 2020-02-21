package messager

import (
	"encoding/hex"
	"fmt"

	"github.com/kevinburke/nacl"
	"github.com/kevinburke/nacl/secretbox"
	"github.com/pkg/errors"
)

// Crypt data with a global key + pin
// It provides basic AES encryption for data
// needed to prevent storing it naked form even in_memory storage
type Crypt struct {
	Key string
}

// Request for both Encrypt and Decrypt
type Request struct {
	Pin  string
	Data []byte
}

// Encrypt to hex with secretbox
func (c Crypt) Encrypt(req Request) ([]byte, error) {

	if len(c.Key)+len(req.Pin) != 32 {
		return nil, errors.New("key+pin should be 32 bytes")
	}
	key, err := nacl.Load(hex.EncodeToString([]byte(fmt.Sprintf("%s%s", c.Key, req.Pin))))
	if err != nil {
		return nil, errors.Wrap(err, "can't make encryption key")
	}
	return secretbox.EasySeal(req.Data, key), nil
}

// Decrypt from hex with secretbox
func (c Crypt) Decrypt(req Request) ([]byte, error) {

	if len(c.Key)+len(req.Pin) != 32 {
		return nil, errors.New("key+pin should be 32 bytes")
	}
	key, err := nacl.Load(hex.EncodeToString([]byte(fmt.Sprintf("%s%s", c.Key, req.Pin))))
	if err != nil {
		return nil, errors.Wrap(err, "can't make decryption key")
	}

	decrypted, err := secretbox.EasyOpen(req.Data, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt")
	}
	return decrypted, nil
}

// MakeSignKey creates 32-pin bytes signKey for AES256
func MakeSignKey(signKey string, pinSize int) (result string) {
	if len(signKey) >= 32-pinSize {
		return signKey[:32-pinSize]
	}

	for i := 0; i <= (32-pinSize)/len(signKey); i++ {
		result += signKey
	}
	return result[:32-pinSize]
}
