package messager

import (
	"fmt"
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

func TestMessageProc_MakeMessage_Errors(t *testing.T) {
	s := &EngineMock{}
	c := &CrypterMock{}

	type args struct {
		duration time.Duration
		pin      string
	}

	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{
			name:    "bad pin error",
			wantErr: ErrBadPin,
			args: args{
				duration: time.Second * 30,
				pin:      "",
			},
		},
		{
			name:    "bad duration",
			wantErr: ErrDuration,
			args: args{
				duration: time.Minute * 30,
				pin:      "1234",
			},
		},
		{
			name:    "internal error when pin too long",
			wantErr: ErrInternal,
			args: args{
				duration: time.Second * 30,
				pin:      "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed gravida varius nisi, id cursus justo. Nulla facilities. Sed auctor, ex eget bibendum aliquet, nisi turpis",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
			r, err := m.MakeMessage(tt.args.duration, "message", tt.args.pin)
			t.Logf("%+v", r)
			require.EqualError(t, err, tt.wantErr.Error())

			assert.Equal(t, 0, len(s.SaveCalls()))
			assert.Equal(t, 0, len(c.EncryptCalls()))
		})
	}
}

func TestMessageProc_MakeMessage_CrypterError(t *testing.T) {
	s := &EngineMock{}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) {
			return nil, fmt.Errorf("crypter error")
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.MakeMessage(time.Second*30, "message", "56789")
	t.Logf("%+v", r)
	require.EqualError(t, err, "crypto error")

	assert.Equal(t, 0, len(s.SaveCalls()))
	assert.Equal(t, 1, len(c.EncryptCalls()))
}

func TestMessageProc_LoadMessage_Err(t *testing.T) {
	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return nil, fmt.Errorf("load error")
		},
	}
	c := &CrypterMock{}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "56789")
	t.Logf("%+v", r)

	require.EqualError(t, err, "load error")

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 0, len(s.RemoveCalls()))
	assert.Equal(t, 0, len(s.IncErrCalls()))
	assert.Equal(t, 0, len(c.DecryptCalls()))
}

func TestMessageProc_LoadMessage_ExpiredErr(t *testing.T) {
	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return &store.Message{
				Data:    []byte("data"),
				PinHash: "some-hash",
				Exp:     time.Now().Add(-1 * time.Minute),
			}, nil
		},
		RemoveFunc: func(key string) error {
			return nil
		},
	}

	c := &CrypterMock{}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "56789")
	t.Logf("%+v", r)

	require.EqualError(t, err, "message expired")

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 1, len(s.RemoveCalls()))
	assert.Equal(t, 0, len(s.IncErrCalls()))
	assert.Equal(t, 0, len(c.DecryptCalls()))
}

func TestMessageProc_LoadMessage_BadPin(t *testing.T) {
	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return &store.Message{
				Data:    []byte("data"),
				PinHash: "some-hash",
				Exp:     time.Now().Add(1 * time.Minute),
			}, nil
		},

		IncErrFunc: func(key string) (int, error) {
			return 0, fmt.Errorf("inc error")
		},
	}

	c := &CrypterMock{}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "56789")
	t.Logf("%+v", r)

	require.EqualError(t, err, "wrong pin")

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 0, len(s.RemoveCalls()))
	assert.Equal(t, 1, len(s.IncErrCalls()))
	assert.Equal(t, 0, len(c.DecryptCalls()))
}

func TestMessageProc_LoadMessage_BadPin_MaxAttempts(t *testing.T) {
	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return &store.Message{
				Data:    []byte("data"),
				PinHash: "some-hash",
				Exp:     time.Now().Add(1 * time.Minute),
			}, nil
		},
		RemoveFunc: func(key string) error {
			return nil
		},

		IncErrFunc: func(key string) (int, error) {
			return 2, nil
		},
	}

	c := &CrypterMock{}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "56789")
	t.Logf("%+v", r)

	require.EqualError(t, err, "wrong pin")

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 1, len(s.RemoveCalls()))
	assert.Equal(t, 1, len(s.IncErrCalls()))
	assert.Equal(t, 0, len(c.DecryptCalls()))
}

func TestMessageProc_LoadMessage_BadPinAttempt(t *testing.T) {
	msg := &store.Message{
		Data:    []byte("data"),
		PinHash: "some-hash",
		Exp:     time.Now().Add(1 * time.Minute),
	}

	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return msg, nil
		},
		IncErrFunc: func(key string) (int, error) {
			return 1, nil
		},
	}

	c := &CrypterMock{}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "56789")
	t.Logf("%+v", r)

	require.EqualError(t, err, "wrong pin attempt")

	assert.Equal(t, r, msg)

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 0, len(s.RemoveCalls()))
	assert.Equal(t, 1, len(s.IncErrCalls()))
	assert.Equal(t, 0, len(c.DecryptCalls()))
}

func TestMessageProc_LoadMessage_DecryptError(t *testing.T) {
	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return &store.Message{
				Data:    []byte("data"),
				PinHash: "$2a$10$2d9OIFG2.zuVIiZznlpy/uJoTl4quQPbDSFnHbi0LuYDILuxHYkDu",
				Exp:     time.Now().Add(1 * time.Minute),
			}, nil
		},
		RemoveFunc: func(key string) error {
			return nil
		},
	}

	c := &CrypterMock{
		DecryptFunc: func(req Request) ([]byte, error) {
			return nil, fmt.Errorf("decrypt error")
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "123456")
	t.Logf("%+v", r)

	require.EqualError(t, err, "wrong pin")

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 1, len(s.RemoveCalls()))
	assert.Equal(t, 0, len(s.IncErrCalls()))
	assert.Equal(t, 1, len(c.DecryptCalls()))
}

func TestMessageProc_LoadMessage(t *testing.T) {
	s := &EngineMock{
		LoadFunc: func(key string) (*store.Message, error) {
			return &store.Message{
				Data:    []byte("data"),
				PinHash: "$2a$10$2d9OIFG2.zuVIiZznlpy/uJoTl4quQPbDSFnHbi0LuYDILuxHYkDu",
				Exp:     time.Now().Add(1 * time.Minute),
			}, nil
		},
		RemoveFunc: func(key string) error {
			return nil
		},
	}

	c := &CrypterMock{
		DecryptFunc: func(req Request) ([]byte, error) {
			return []byte("decrypted blah"), nil
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute})
	r, err := m.LoadMessage("somekey", "123456")
	t.Logf("%+v", r)

	require.NoError(t, err)
	assert.Equal(t, "decrypted blah", string(r.Data))
	assert.Equal(t, 0, r.Errors)
	assert.Contains(t, r.PinHash, "$2a$")

	assert.Equal(t, 1, len(s.LoadCalls()))
	assert.Equal(t, 1, len(s.RemoveCalls()))
	assert.Equal(t, 0, len(s.IncErrCalls()))
	assert.Equal(t, 1, len(c.DecryptCalls()))
}
