package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	s := NewInMemory(time.Second)
	msg := Message{Key: "key123456", Exp: time.Now(), Data: []byte("data string"), PinHash: "123456"}
	require.NoError(t, s.Save(&msg), "saved fine")
	savedMsg, err := s.Load("key123456")
	require.NoError(t, err, "key found")
	assert.Equal(t, msg, *savedMsg, "matches loaded msg")
	_, err = s.Load("badkey123456")
	assert.Equal(t, ErrLoadRejected, err, "no such key")
}

func TestIncErr(t *testing.T) {
	s := NewInMemory(time.Second)
	msg := Message{Key: "key123456", Exp: time.Now(), Data: []byte("data string"), PinHash: "123456"}
	require.NoError(t, s.Save(&msg))

	cnt, err := s.IncErr("key123456")
	require.NoError(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = s.IncErr("key123456")
	require.NoError(t, err)
	assert.Equal(t, 2, cnt)

	_, err = s.IncErr("aaakey123456")
	assert.Equal(t, ErrLoadRejected, err)
}

func TestCleaner(t *testing.T) {
	s := NewInMemory(time.Millisecond * 50)
	msg := Message{Key: "key123456", Exp: time.Now(), Data: []byte("data string"), PinHash: "123456"}
	require.NoError(t, s.Save(&msg), "saved fine")

	_, err := s.Load("key123456")
	require.NoError(t, err, "key still in store")
	time.Sleep(time.Millisecond * 101)

	_, err = s.Load("key123456")
	assert.Equal(t, ErrLoadRejected, err, "msg gone")
}
