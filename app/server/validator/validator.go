// Package validator provides functionality for validating and sanitizing data.
package validator

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Validator is a struct that contains  field errors and a non-field errors.
type Validator struct {
	FieldErrors    map[string]string
	NonFieldErrors []string
}

// Valid returns true if the FieldErrors map is empty, otherwise false.
func (v *Validator) Valid() bool {
	return len(v.FieldErrors) == 0
}

// AddFieldError adds an error message to the FieldErrors map.
func (v *Validator) AddFieldError(key, message string) {
	if v.FieldErrors == nil {
		v.FieldErrors = make(map[string]string)
	}

	if _, exists := v.FieldErrors[key]; !exists {
		v.FieldErrors[key] = message
	}
}

// AddNonFieldError adds an error message to the NonFieldErrors slice.
func (v *Validator) AddNonFieldError(message string) {
	v.NonFieldErrors = append(v.NonFieldErrors, message)
}

// CheckField adds an error message to the FieldErrors map only if a  validation check is not passed.
func (v *Validator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

// NotBlank returns true if a value is not an empty string.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// Blank returns true if a value is an empty string.
func Blank(value string) bool {
	return strings.TrimSpace(value) == ""
}

// MaxChars returns true if a value contains no more than n characters.
func MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(strings.TrimSpace(value)) <= n
}

// MinChars returns true if a value contains at least n characters.
func MinChars(value string, n int) bool {
	return utf8.RuneCountInString(strings.TrimSpace(value)) >= n
}

// IsNumber returns true if specified value is a number.
func IsNumber(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

// MaxDuration validates if a duration is within the maximum allowed duration.
func MaxDuration(d, maxDuration time.Duration) bool {
	return d <= maxDuration
}

// PermittedValue returns true if a value is in a list of permitted integers.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	for i := range permittedValues {
		if value == permittedValues[i] {
			return true
		}

	}
	return false
}
