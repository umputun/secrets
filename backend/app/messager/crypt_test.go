package messager

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			data: "abcdefg something 12345 ?? what?",
			pin:  "00000",
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
			err:  fmt.Errorf("key+pin should be 32 bytes, got 31"),
		},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r1, err := c.Encrypt(Request{Data: []byte(tt.data), Pin: tt.pin})
			if tt.err != nil {
				require.EqualError(t, err, tt.err.Error())
				return
			}
			assert.NoError(t, err)
			t.Logf("%x", r1)

			r2, err := c.Decrypt(Request{Data: r1, Pin: tt.pin})
			assert.NoError(t, err)
			assert.Equal(t, tt.data, string(r2))
		})
	}
}

func TestMakeSignKey(t *testing.T) {

	tbl := []struct {
		key     string
		pinSize int
		res     string
	}{
		{
			key:     "abcdefg",
			pinSize: 5,
			res:     "abcdefgabcdefgabcdefgabcdef",
		},
		{
			key:     "abcdefgabcdefgabcdefgabcdef",
			pinSize: 5,
			res:     "abcdefgabcdefgabcdefgabcdef",
		},
		{
			key:     "11223344556677889900112233445566778899001122334455",
			pinSize: 6,
			res:     "11223344556677889900112233",
		},
	}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.res, MakeSignKey(tt.key, tt.pinSize))
			assert.Equal(t, 32, len(tt.res)+tt.pinSize)
		})
	}
}
