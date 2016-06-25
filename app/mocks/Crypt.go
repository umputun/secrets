package mocks

import "github.com/umputun/secrets/app/crypt"
import "github.com/stretchr/testify/mock"

type Crypt struct {
	mock.Mock
}

func (_m *Crypt) Encrypt(req crypt.Request) (string, error) {
	ret := _m.Called(req)

	var r0 string
	if rf, ok := ret.Get(0).(func(crypt.Request) string); ok {
		r0 = rf(req)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(crypt.Request) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Crypt) Decrypt(req crypt.Request) (string, error) {
	ret := _m.Called(req)

	var r0 string
	if rf, ok := ret.Get(0).(func(crypt.Request) string); ok {
		r0 = rf(req)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(crypt.Request) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
