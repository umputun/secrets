// Package crypt provides basic AES encryption for data
// needed to prevent storing it naked even in_memory storage
package crypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

// Crypt data with a global key + pin
type Crypt struct {
	Key string
}

// Request for both Encrypt and Decrypt
type Request struct {
	Pin  string
	Data string
}

// Encrypt to hex with AES256
func (c Crypt) Encrypt(req Request) (result string, err error) {

	if len(c.Key)+len(req.Pin) != 32 {
		return "", fmt.Errorf("key+pin should be 32 bytes")
	}
	key := []byte(fmt.Sprintf("%s%s", c.Key, req.Pin))

	var block cipher.Block

	if block, err = aes.NewCipher(key); err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(req.Data))

	// iv =  initialization vector
	iv := ciphertext[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(req.Data))

	hexRes := make([]byte, hex.EncodedLen(len(ciphertext)))
	hex.Encode(hexRes, ciphertext)
	return string(hexRes), nil
}

// Decrypt from hex with AES256
func (c Crypt) Decrypt(req Request) (result string, err error) {

	if len(c.Key)+len(req.Pin) != 32 {
		return "", fmt.Errorf("key+pin should be 32 bytes")
	}
	key := []byte(fmt.Sprintf("%s%s", c.Key, req.Pin))

	var block cipher.Block

	if block, err = aes.NewCipher(key); err != nil {
		return
	}

	data := make([]byte, hex.DecodedLen(len(req.Data)))
	hex.Decode(data, []byte(req.Data))
	if len(data) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := []byte(data[:aes.BlockSize])
	ciphertext := []byte(data[aes.BlockSize:])

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
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
