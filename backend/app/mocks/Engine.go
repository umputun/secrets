package mocks

import "github.com/umputun/secrets/backend/app/store"
import "github.com/stretchr/testify/mock"

type Engine struct {
	mock.Mock
}

func (_m *Engine) Save(msg *store.Message) error {
	ret := _m.Called(msg)

	var r0 error
	if rf, ok := ret.Get(0).(func(*store.Message) error); ok {
		r0 = rf(msg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Engine) Load(key string) (*store.Message, error) {
	ret := _m.Called(key)

	var r0 *store.Message
	if rf, ok := ret.Get(0).(func(string) *store.Message); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*store.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Engine) IncErr(key string) (int, error) {
	ret := _m.Called(key)

	var r0 int
	if rf, ok := ret.Get(0).(func(string) int); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Engine) Remove(key string) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
