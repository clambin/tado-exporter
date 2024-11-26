// Code generated by mockery v2.49.0. DO NOT EDIT.

package mocks

import (
	poller "github.com/clambin/tado-exporter/internal/poller"
	mock "github.com/stretchr/testify/mock"
)

// Poller is an autogenerated mock type for the Poller type
type Poller struct {
	mock.Mock
}

type Poller_Expecter struct {
	mock *mock.Mock
}

func (_m *Poller) EXPECT() *Poller_Expecter {
	return &Poller_Expecter{mock: &_m.Mock}
}

// Refresh provides a mock function with given fields:
func (_m *Poller) Refresh() {
	_m.Called()
}

// Poller_Refresh_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Refresh'
type Poller_Refresh_Call struct {
	*mock.Call
}

// Refresh is a helper method to define mock.On call
func (_e *Poller_Expecter) Refresh() *Poller_Refresh_Call {
	return &Poller_Refresh_Call{Call: _e.mock.On("Refresh")}
}

func (_c *Poller_Refresh_Call) Run(run func()) *Poller_Refresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Poller_Refresh_Call) Return() *Poller_Refresh_Call {
	_c.Call.Return()
	return _c
}

func (_c *Poller_Refresh_Call) RunAndReturn(run func()) *Poller_Refresh_Call {
	_c.Call.Return(run)
	return _c
}

// Subscribe provides a mock function with given fields:
func (_m *Poller) Subscribe() chan poller.Update {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Subscribe")
	}

	var r0 chan poller.Update
	if rf, ok := ret.Get(0).(func() chan poller.Update); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan poller.Update)
		}
	}

	return r0
}

// Poller_Subscribe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Subscribe'
type Poller_Subscribe_Call struct {
	*mock.Call
}

// Subscribe is a helper method to define mock.On call
func (_e *Poller_Expecter) Subscribe() *Poller_Subscribe_Call {
	return &Poller_Subscribe_Call{Call: _e.mock.On("Subscribe")}
}

func (_c *Poller_Subscribe_Call) Run(run func()) *Poller_Subscribe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Poller_Subscribe_Call) Return(_a0 chan poller.Update) *Poller_Subscribe_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Poller_Subscribe_Call) RunAndReturn(run func() chan poller.Update) *Poller_Subscribe_Call {
	_c.Call.Return(run)
	return _c
}

// Unsubscribe provides a mock function with given fields: ch
func (_m *Poller) Unsubscribe(ch chan poller.Update) {
	_m.Called(ch)
}

// Poller_Unsubscribe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Unsubscribe'
type Poller_Unsubscribe_Call struct {
	*mock.Call
}

// Unsubscribe is a helper method to define mock.On call
//   - ch chan poller.Update
func (_e *Poller_Expecter) Unsubscribe(ch interface{}) *Poller_Unsubscribe_Call {
	return &Poller_Unsubscribe_Call{Call: _e.mock.On("Unsubscribe", ch)}
}

func (_c *Poller_Unsubscribe_Call) Run(run func(ch chan poller.Update)) *Poller_Unsubscribe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(chan poller.Update))
	})
	return _c
}

func (_c *Poller_Unsubscribe_Call) Return() *Poller_Unsubscribe_Call {
	_c.Call.Return()
	return _c
}

func (_c *Poller_Unsubscribe_Call) RunAndReturn(run func(chan poller.Update)) *Poller_Unsubscribe_Call {
	_c.Call.Return(run)
	return _c
}

// NewPoller creates a new instance of Poller. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPoller(t interface {
	mock.TestingT
	Cleanup(func())
}) *Poller {
	mock := &Poller{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
