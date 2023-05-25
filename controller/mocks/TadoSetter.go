// Code generated by mockery v2.28.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// TadoSetter is an autogenerated mock type for the TadoSetter type
type TadoSetter struct {
	mock.Mock
}

// DeleteZoneOverlay provides a mock function with given fields: _a0, _a1
func (_m *TadoSetter) DeleteZoneOverlay(_a0 context.Context, _a1 int) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetZoneOverlay provides a mock function with given fields: _a0, _a1, _a2
func (_m *TadoSetter) SetZoneOverlay(_a0 context.Context, _a1 int, _a2 float64) error {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, float64) error); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetZoneTemporaryOverlay provides a mock function with given fields: _a0, _a1, _a2, _a3
func (_m *TadoSetter) SetZoneTemporaryOverlay(_a0 context.Context, _a1 int, _a2 float64, _a3 time.Duration) error {
	ret := _m.Called(_a0, _a1, _a2, _a3)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, int, float64, time.Duration) error); ok {
		r0 = rf(_a0, _a1, _a2, _a3)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewTadoSetter interface {
	mock.TestingT
	Cleanup(func())
}

// NewTadoSetter creates a new instance of TadoSetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewTadoSetter(t mockConstructorTestingTNewTadoSetter) *TadoSetter {
	mock := &TadoSetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
