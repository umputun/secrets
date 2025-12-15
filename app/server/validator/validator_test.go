package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidator_Valid(t *testing.T) {
	type args struct {
		FieldErrors    map[string]string
		NonFieldErrors []string
	}
	tests := []struct {
		name       string
		args       args
		wantResult bool
	}{
		{
			name: "Valid when errors are nil",
			args: args{
				FieldErrors:    nil,
				NonFieldErrors: nil,
			},
			wantResult: true,
		},
		{
			name: "Valid when errors are empty",
			args: args{
				FieldErrors:    make(map[string]string),
				NonFieldErrors: []string{},
			},
			wantResult: true,
		},
		{
			name: "Invalid when field errors are not empty",
			args: args{
				FieldErrors: map[string]string{
					"email": "email is required",
				},
				NonFieldErrors: []string{},
			},
			wantResult: false,
		},
		{
			name: "Valid when non-field errors are not empty",
			args: args{
				FieldErrors:    make(map[string]string),
				NonFieldErrors: []string{"invalid request"},
			},
			wantResult: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{
				FieldErrors:    tt.args.FieldErrors,
				NonFieldErrors: tt.args.NonFieldErrors,
			}
			r := v.Valid()

			assert.Equal(t, tt.wantResult, r)
		})
	}
}

func TestValidator_AddFieldError(t *testing.T) {
	type args struct {
		key     string
		message string
	}
	tests := []struct {
		name       string
		args       args
		wantResult bool
		wantKey    string
	}{
		{
			name: "Add field error",
			args: args{
				key:     "email",
				message: "email is required",
			},
			wantResult: false,
		},
		{
			name: "Don't add field error when key already exists",
			args: args{
				key:     "email",
				message: "email is required",
			},
			wantResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{}
			v.AddFieldError(tt.args.key, tt.args.message)

			assert.Equal(t, tt.wantResult, v.Valid())
			assert.Equal(t, tt.args.message, v.FieldErrors[tt.args.key])
		})
	}
}

func TestValidator_AddNonFieldError(t *testing.T) {
	message := "invalid request"

	v := &Validator{}
	v.AddNonFieldError(message)

	assert.Equal(t, message, v.NonFieldErrors[0])
}

func TestValidator_CheckField(t *testing.T) {
	type args struct {
		ok      bool
		key     string
		message string
	}
	tests := []struct {
		name       string
		args       args
		wantResult bool
		wantMsg    string
	}{
		{
			name: "Add field error when check is not ok",
			args: args{
				ok:      false,
				key:     "email",
				message: "email is required",
			},
			wantResult: false,
			wantMsg:    "email is required",
		},
		{
			name: "Don't add field error when check is ok",
			args: args{
				ok:      true,
				key:     "email",
				message: "email is required",
			},
			wantResult: true,
			wantMsg:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{}
			v.CheckField(tt.args.ok, tt.args.key, tt.args.message)

			assert.Equal(t, tt.wantResult, v.Valid())
			assert.Equal(t, tt.wantMsg, v.FieldErrors[tt.args.key])
		})
	}
}

func TestValidator_NotBlank(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantResult bool
	}{
		{
			name:       "NotBlank returns false for empty string",
			value:      "",
			wantResult: false,
		},
		{
			name:       "NotBlank returns false for string with only spaces",
			value:      "   ",
			wantResult: false,
		},
		{
			name:       "NotBlank returns false for string with only spaces",
			value:      "\n\t  \n",
			wantResult: false,
		},
		{
			name:       "NotBlank returns true for string with non-space characters",
			value:      "abc",
			wantResult: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NotBlank(tt.value)

			assert.Equal(t, tt.wantResult, r)
		})
	}
}

func TestValidator_MaxChars(t *testing.T) {
	type args struct {
		value string
		n     int
	}
	tests := []struct {
		name       string
		args       args
		wantResult bool
	}{
		{
			name: "MaxChars returns false for string with more than n characters",
			args: args{
				value: "abc",
				n:     2,
			},
			wantResult: false,
		},
		{
			name: "MaxChars returns true for string with less than n characters",
			args: args{
				value: "abc",
				n:     3,
			},
			wantResult: true,
		},
		{
			name: "MaxChars returns true for empty string",
			args: args{
				value: "",
				n:     3,
			},
			wantResult: true,
		},
		{
			name: "MaxChars returns true for string with only spaces",
			args: args{
				value: "   ",
				n:     3,
			},
			wantResult: true,
		},
		{
			name: "MaxChars returns true for string with only spaces",
			args: args{
				value: "\n\t  \n",
				n:     3,
			},
			wantResult: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := MaxChars(tt.args.value, tt.args.n)

			assert.Equal(t, tt.wantResult, r)
		})
	}
}

func TestValidator_IsNumber(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantResult bool
	}{
		{
			name:       "IsNumber returns false for non-number string",
			value:      "abc",
			wantResult: false,
		},
		{
			name:       "IsNumber returns true for number string",
			value:      "123",
			wantResult: true,
		},
		{
			name:       "IsNumber returns false for empty string",
			value:      "",
			wantResult: false,
		},
		{
			name:       "IsNumber returns false for string with only spaces",
			value:      "   ",
			wantResult: false,
		},
		{
			name:       "IsNumber returns false for string with only spaces",
			value:      "\n\t  \n",
			wantResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := IsNumber(tt.value)

			assert.Equal(t, tt.wantResult, r)
		})
	}
}

func TestValidator_MaxDuration(t *testing.T) {
	type args struct {
		d           time.Duration
		maxDuration time.Duration
	}
	tests := []struct {
		name       string
		args       args
		wantResult bool
	}{
		{
			name: "MaxDuration returns false for duration greater than max duration",
			args: args{
				d:           time.Hour,
				maxDuration: time.Minute,
			},
			wantResult: false,
		},
		{
			name: "MaxDuration returns true for duration less than max duration",
			args: args{
				d:           time.Minute,
				maxDuration: time.Hour,
			},
			wantResult: true,
		},
		{
			name: "MaxDuration returns true for duration equal to max duration",
			args: args{
				d:           time.Minute,
				maxDuration: time.Minute,
			},
			wantResult: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := MaxDuration(tt.args.d, tt.args.maxDuration)

			assert.Equal(t, tt.wantResult, r)
		})
	}
}

func TestValidator_MinChars(t *testing.T) {
	type args struct {
		value string
		n     int
	}
	tests := []struct {
		name       string
		args       args
		wantResult bool
	}{
		{
			name: "MinChars returns false for string with less than n characters",
			args: args{
				value: "abc",
				n:     4,
			},
			wantResult: false,
		},
		{
			name: "MinChars returns true for string with more than n characters",
			args: args{
				value: "abc",
				n:     2,
			},
			wantResult: true,
		},
		{
			name: "MinChars returns true for string with n characters",
			args: args{
				value: "abc",
				n:     3,
			},
			wantResult: true,
		},
		{
			name: "MinChars returns false for empty string",
			args: args{
				value: "",
				n:     3,
			},
			wantResult: false,
		},
		{
			name: "MinChars returns false for string with only spaces",
			args: args{
				value: "   ",
				n:     3,
			},
			wantResult: false,
		},
		{
			name: "MinChars returns false for string with only spaces",
			args: args{
				value: "\n\t  \n",
				n:     3,
			},
			wantResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := MinChars(tt.args.value, tt.args.n)

			assert.Equal(t, tt.wantResult, r)
		})
	}
}
