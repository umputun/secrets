package messager

import (
	"fmt"
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
	assert.Equal(t, int64(1024*1024), m.MaxFileSize)
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
		SaveFunc: func(msg *store.Message) error { return nil },
	}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) { return []byte("encrypted-data"), nil },
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: 1024})
	r, err := m.MakeFileMessage(FileRequest{
		Duration:    time.Second * 30,
		Pin:         "12345",
		FileName:    "test.pdf",
		ContentType: "application/pdf",
		Data:        []byte("file content"),
	})

	require.NoError(t, err)
	assert.True(t, IsFileMessage(r.Data))

	filename, contentType, dataStart := ParseFileHeader(r.Data)
	assert.Equal(t, "test.pdf", filename)
	assert.Equal(t, "application/pdf", contentType)
	assert.Equal(t, "encrypted-data", string(r.Data[dataStart:]))
	assert.Contains(t, r.PinHash, "$2a$")

	assert.Len(t, s.SaveCalls(), 1)
	assert.Len(t, c.EncryptCalls(), 1)
}

func TestMessageProc_MakeFileMessage_Errors(t *testing.T) {
	tests := []struct {
		name    string
		req     FileRequest
		wantErr error
	}{
		{name: "empty pin", req: FileRequest{Duration: time.Second, Pin: "", FileName: "f.txt", ContentType: "text/plain", Data: []byte("x")}, wantErr: ErrBadPin},
		{name: "empty filename", req: FileRequest{Duration: time.Second, Pin: "123", FileName: "", ContentType: "text/plain", Data: []byte("x")}, wantErr: ErrBadFileName},
		{name: "filename too long", req: FileRequest{Duration: time.Second, Pin: "123", FileName: string(make([]byte, 256)), ContentType: "text/plain", Data: []byte("x")}, wantErr: ErrBadFileName},
		{name: "file too large", req: FileRequest{Duration: time.Second, Pin: "123", FileName: "f.txt", ContentType: "text/plain", Data: make([]byte, 2048)}, wantErr: ErrFileTooLarge},
		{name: "bad duration", req: FileRequest{Duration: time.Hour, Pin: "123", FileName: "f.txt", ContentType: "text/plain", Data: []byte("x")}, wantErr: ErrDuration},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &EngineMock{}
			c := &CrypterMock{}
			m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: 1024})
			_, err := m.MakeFileMessage(tt.req)
			require.EqualError(t, err, tt.wantErr.Error())
			assert.Empty(t, s.SaveCalls())
		})
	}
}

func TestMessageProc_MakeFileMessage_CrypterError(t *testing.T) {
	s := &EngineMock{}
	c := &CrypterMock{
		EncryptFunc: func(req Request) ([]byte, error) { return nil, fmt.Errorf("encrypt error") },
	}

	m := New(s, c, Params{MaxPinAttempts: 2, MaxDuration: time.Minute, MaxFileSize: 1024})
	_, err := m.MakeFileMessage(FileRequest{Duration: time.Second, Pin: "123", FileName: "f.txt", ContentType: "text/plain", Data: []byte("x")})
	require.EqualError(t, err, ErrCrypto.Error())
	assert.Empty(t, s.SaveCalls())
	assert.Len(t, c.EncryptCalls(), 1)
}

func TestIsFileMessage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "valid file message", data: []byte("!!FILE!!test.pdf!!application/pdf!!\ndata"), want: true},
		{name: "text message", data: []byte("encrypted text"), want: false},
		{name: "empty data", data: []byte{}, want: false},
		{name: "partial prefix", data: []byte("!!FIL"), want: false},
		{name: "just prefix", data: []byte("!!FILE!!"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsFileMessage(tt.data))
		})
	}
}

func TestParseFileHeader(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantName    string
		wantType    string
		wantStart   int
		wantInvalid bool
	}{
		{name: "valid header", data: []byte("!!FILE!!test.pdf!!application/pdf!!\nencrypted"), wantName: "test.pdf", wantType: "application/pdf", wantStart: 36},
		{name: "empty filename", data: []byte("!!FILE!!!!text/plain!!\ndata"), wantName: "", wantType: "text/plain", wantStart: 23},
		{name: "not a file", data: []byte("encrypted text"), wantInvalid: true},
		{name: "no newline", data: []byte("!!FILE!!test.pdf!!application/pdf!!"), wantInvalid: true},
		{name: "missing content type", data: []byte("!!FILE!!test.pdf!!!!\ndata"), wantName: "test.pdf", wantType: "", wantStart: 21},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ctype, start := ParseFileHeader(tt.data)
			if tt.wantInvalid {
				assert.Equal(t, -1, start)
				return
			}
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantType, ctype)
			assert.Equal(t, tt.wantStart, start)
		})
	}
}
