package crypt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrypt(t *testing.T) {
	c := Crypt{Key: "123456789012345678901234567"}

	tbl := []struct {
		data string
		pin  string
		err  error
	}{
		{
			data: "abcdefg",
			pin:  "99999",
		},
		{
			data: "abcdefg",
			pin:  "12345",
		},
		{
			data: "",
			pin:  "12345",
		},
		{
			data: "dfasdfasd asdfasdfa asdfasdf asdfasdfasdf asdfasdf",
			pin:  "abcde",
		},
		{
			data: "dfasdfasd asdfasdfa asdfasdf asdfasdfasdf asdfasdf",
			pin:  "abcd",
			err:  fmt.Errorf("key+pin should be 32 bytes"),
		},
	}

	for _, tt := range tbl {
		r1, err := c.Encrypt(Request{Data: tt.data, Pin: tt.pin})
		if tt.err != nil {
			assert.Equal(t, tt.err, err)
			continue
		}
		assert.Nil(t, err)
		t.Logf("%s", r1)

		r2, err := c.Decrypt(Request{Data: r1, Pin: tt.pin})
		assert.Nil(t, err)
		assert.Equal(t, tt.data, r2)
	}
}
