package messager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/secrets/backend/app/store"
)

func TestMessageProc_NewDefault(t *testing.T) {
	m := New(nil, Crypt{}, Params{})
	assert.Equal(t, time.Hour*24*31, m.Params.MaxDuration)
	assert.Equal(t, 3, m.Params.MaxPinAttempts)
}

func TestMessageProc_MakeMessage(t *testing.T) {
	s := &EngineMock{
		SaveFunc: func(msg *store.Message) error {
			return nil
		},
	}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) {
			return []byte("encrypted blah"), nil
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.MakeMessage(time.Second*30, "message", "56789")
	t.Logf("%+v", r)
	require.NoError(t, err)
	assert.Equal(t, "encrypted blah", string(r.Data))
	assert.Equal(t, 0, r.Errors)
	assert.Contains(t, r.PinHash, "$2a$")

	assert.Equal(t, 1, len(s.SaveCalls()))
	assert.Equal(t, 1, len(c.EncryptCalls()))
}

// func TestMessageProc_LoadMessage(t *testing.T) {
// 	s := &engine.MockEngine{}
// 	c := &MockCrypter{}
// 	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
//
// 	pinHash, err := m.makeHash("56789")
// 	require.NoError(t, err)
//
// 	s.On("Load", mock.AnythingOfType("string")).Return(&engine.Message{
// 		Data:    []byte("data"),
// 		PinHash: pinHash,
// 		Key:     "somekey",
// 		Exp:     time.Now().Add(time.Minute),
// 	}, nil)
// 	s.On("Remove", "somekey").Return(nil)
// 	c.On("Decrypt", mock.Anything).Return([]byte("decrypted blah"), nil)
//
// 	r, err := m.LoadMessage("somekey", "56789")
// 	t.Logf("%+v", r)
// 	require.NoError(t, err)
// 	assert.Equal(t, "decrypted blah", string(r.Data))
// 	assert.Equal(t, 0, r.Errors)
// 	assert.Contains(t, r.PinHash, "$2a$")
//
// 	s.AssertExpectations(t)
// 	c.AssertExpectations(t)
// }
//
// func TestMessageProc_BadPin(t *testing.T) {
// 	s := &engine.MockEngine{}
// 	c := &MockCrypter{}
// 	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
//
// 	s.On("Load", mock.AnythingOfType("string")).Return(&engine.Message{
// 		Data:    []byte("data"),
// 		PinHash: "bad bad",
// 		Key:     "somekey",
// 		Exp:     time.Now().Add(time.Minute),
// 	}, nil)
// 	s.On("IncErr", "somekey").Return(1, nil).Times(1)
// 	s.On("IncErr", "somekey").Return(2, nil).Times(1)
// 	s.On("Remove", "somekey").Return(nil).Times(1)
//
// 	_, err := m.LoadMessage("somekey", "56789")
// 	require.EqualError(t, err, "wrong pin attempt")
// 	_, err = m.LoadMessage("somekey", "56789")
// 	require.EqualError(t, err, "wrong pin")
//
// 	s.AssertExpectations(t)
// }
