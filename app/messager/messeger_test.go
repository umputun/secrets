package messager

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/secrets/app/store"
)

func TestMessageProc_NewDefault(t *testing.T) {
	m := New(nil, Crypt{}, Params{})
	assert.Equal(t, time.Hour*24*31, m.MaxDuration)
	assert.Equal(t, 3, m.MaxPinAttempts)
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

	assert.Len(t, s.SaveCalls(), 1)
	assert.Len(t, c.EncryptCalls(), 1)
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

			assert.Empty(t, s.SaveCalls())
			assert.Empty(t, c.EncryptCalls())
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

	assert.Empty(t, s.SaveCalls())
	assert.Len(t, c.EncryptCalls(), 1)
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Empty(t, s.RemoveCalls())
	assert.Empty(t, s.IncErrCalls())
	assert.Empty(t, c.DecryptCalls())
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Len(t, s.RemoveCalls(), 1)
	assert.Empty(t, s.IncErrCalls())
	assert.Empty(t, c.DecryptCalls())
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Empty(t, s.RemoveCalls())
	assert.Len(t, s.IncErrCalls(), 1)
	assert.Empty(t, c.DecryptCalls())
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Len(t, s.RemoveCalls(), 1)
	assert.Len(t, s.IncErrCalls(), 1)
	assert.Empty(t, c.DecryptCalls())
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Empty(t, s.RemoveCalls())
	assert.Len(t, s.IncErrCalls(), 1)
	assert.Empty(t, c.DecryptCalls())
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Len(t, s.RemoveCalls(), 1)
	assert.Empty(t, s.IncErrCalls())
	assert.Len(t, c.DecryptCalls(), 1)
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

	assert.Len(t, s.LoadCalls(), 1)
	assert.Len(t, s.RemoveCalls(), 1)
	assert.Empty(t, s.IncErrCalls())
	assert.Len(t, c.DecryptCalls(), 1)
}

func TestMessageProc_MakeFileMessage(t *testing.T) {
	s := &EngineMock{
		SaveFunc: func(msg *store.Message) error {
			return nil
		},
	}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) {
			return []byte("encrypted file data"), nil
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: 1024})
	r, err := m.MakeFileMessage(time.Second*30, []byte("file content"), "test.txt", "text/plain", "56789")
	t.Logf("%+v", r)
	require.NoError(t, err)
	assert.Equal(t, "encrypted file data", string(r.Data))
	assert.Equal(t, 0, r.Errors)
	assert.Contains(t, r.PinHash, "$2a$")
	assert.True(t, r.IsFile)
	assert.Equal(t, "test.txt", r.FileName)
	assert.Equal(t, "text/plain", r.ContentType)
	assert.Equal(t, int64(12), r.FileSize)

	assert.Len(t, s.SaveCalls(), 1)
	assert.Len(t, c.EncryptCalls(), 1)
}

func TestMessageProc_MakeFileMessage_Errors(t *testing.T) {
	s := &EngineMock{}
	c := &CrypterMock{}

	// create a filename longer than 255 bytes
	longFileName := strings.Repeat("a", 256) + ".txt"

	tests := []struct {
		name        string
		duration    time.Duration
		data        []byte
		fileName    string
		pin         string
		maxFileSize int64
		wantErr     error
	}{
		{name: "empty pin", duration: time.Second * 30, data: []byte("data"), fileName: "test.txt", pin: "", maxFileSize: 1024, wantErr: ErrBadPin},
		{name: "filename too long", duration: time.Second * 30, data: []byte("data"), fileName: longFileName, pin: "1234", maxFileSize: 1024, wantErr: ErrFileNameLength},
		{name: "file too large", duration: time.Second * 30, data: []byte("large data here"), fileName: "test.txt", pin: "1234", maxFileSize: 5, wantErr: ErrFileTooLarge},
		{name: "zero duration", duration: 0, data: []byte("data"), fileName: "test.txt", pin: "1234", maxFileSize: 1024, wantErr: ErrDuration},
		{name: "negative duration", duration: -time.Second, data: []byte("data"), fileName: "test.txt", pin: "1234", maxFileSize: 1024, wantErr: ErrDuration},
		{name: "duration exceeded", duration: time.Minute * 30, data: []byte("data"), fileName: "test.txt", pin: "1234", maxFileSize: 1024, wantErr: ErrDuration},
		{
			name: "pin too long causes hash error", duration: time.Second * 30, data: []byte("data"), fileName: "test.txt", maxFileSize: 1024,
			pin:     "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed gravida varius nisi, id cursus justo. Nulla facilities",
			wantErr: ErrInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: tt.maxFileSize})
			r, err := m.MakeFileMessage(tt.duration, tt.data, tt.fileName, "text/plain", tt.pin)
			t.Logf("%+v", r)
			require.EqualError(t, err, tt.wantErr.Error())

			assert.Empty(t, s.SaveCalls())
			assert.Empty(t, c.EncryptCalls())
		})
	}
}

func TestMessageProc_MakeFileMessage_CrypterError(t *testing.T) {
	s := &EngineMock{}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) {
			return nil, fmt.Errorf("crypter error")
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: 1024})
	r, err := m.MakeFileMessage(time.Second*30, []byte("file data"), "test.txt", "text/plain", "56789")
	t.Logf("%+v", r)
	require.EqualError(t, err, "crypto error")

	assert.Empty(t, s.SaveCalls())
	assert.Len(t, c.EncryptCalls(), 1)
}

func TestMessageProc_MakeFileMessage_NoMaxFileSize(t *testing.T) {
	// test that file size check is skipped when MaxFileSize is 0
	s := &EngineMock{
		SaveFunc: func(msg *store.Message) error {
			return nil
		},
	}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) {
			return []byte("encrypted"), nil
		},
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: 0})
	r, err := m.MakeFileMessage(time.Second*30, []byte("any size file"), "test.txt", "text/plain", "56789")
	require.NoError(t, err)
	assert.True(t, r.IsFile)
	assert.Len(t, s.SaveCalls(), 1)
}
