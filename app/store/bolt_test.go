package store

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveAndLoadBolt(t *testing.T) {
	s, err := NewBolt("/tmp/test.bd", time.Minute)
	assert.Nil(t, err, "engine created")

	msg := Message{Key: "key123456", Exp: time.Now(), Data: "data string", PinHash: "123456"}
	assert.Nil(t, s.Save(&msg), "saved fine")
	savedMsg, err := s.Load("key123456")
	assert.Nil(t, err, "key found")
	assert.EqualValues(t, msg, *savedMsg, "matches loaded msg")
	_, err = s.Load("badkey123456")
	assert.Equal(t, ErrLoadRejected, err, "no such key")

	os.Remove("/tmp/test.bd")
}

func TestIncErrBolt(t *testing.T) {
	s, err := NewBolt("/tmp/test.bd", time.Minute)
	assert.Nil(t, err, "engine created")

	msg := Message{Key: "key123456", Exp: time.Now(), Data: "data string", PinHash: "123456"}
	assert.Nil(t, s.Save(&msg))

	cnt, err := s.IncErr("key123456")
	assert.Nil(t, err)
	assert.Equal(t, 1, cnt)

	cnt, err = s.IncErr("key123456")
	assert.Nil(t, err)
	assert.Equal(t, 2, cnt)

	_, err = s.IncErr("aaakey123456")
	assert.Equal(t, ErrLoadRejected, err)
	os.Remove("/tmp/test.bd")
}

func TestCleanerBolt(t *testing.T) {
	s, err := NewBolt("/tmp/test.bd", time.Millisecond*50)
	assert.Nil(t, err, "engine created")
	exp := time.Now().Add(time.Second)
	msg := Message{Key: fmt.Sprintf("%d-key123456", exp.Unix()), Exp: exp, Data: "data string", PinHash: "123456"}
	assert.Nil(t, s.Save(&msg), "saved fine")

	_, err = s.Load(fmt.Sprintf("%d-key123456", exp.Unix()))
	assert.Nil(t, err, "key still in store")
	time.Sleep(time.Millisecond * 1500)

	_, err = s.Load(fmt.Sprintf("%d-key123456", exp.Unix()))
	assert.Equal(t, ErrLoadRejected, err, "msg gone")
	os.Remove("/tmp/test.bd")
}
