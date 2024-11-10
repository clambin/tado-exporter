// Code generated by mockery v2.46.3. DO NOT EDIT.

package mocks

import (
	context "context"

	slack "github.com/slack-go/slack"
	mock "github.com/stretchr/testify/mock"

	socketmode "github.com/slack-go/slack/socketmode"
)

// SlackApp is an autogenerated mock type for the SlackApp type
type SlackApp struct {
	mock.Mock
}

type SlackApp_Expecter struct {
	mock *mock.Mock
}

func (_m *SlackApp) EXPECT() *SlackApp_Expecter {
	return &SlackApp_Expecter{mock: &_m.Mock}
}

// AddSlashCommand provides a mock function with given fields: _a0, _a1
func (_m *SlackApp) AddSlashCommand(_a0 string, _a1 func(slack.SlashCommand, *socketmode.Client)) {
	_m.Called(_a0, _a1)
}

// SlackApp_AddSlashCommand_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddSlashCommand'
type SlackApp_AddSlashCommand_Call struct {
	*mock.Call
}

// AddSlashCommand is a helper method to define mock.On call
//   - _a0 string
//   - _a1 func(slack.SlashCommand , *socketmode.Client)
func (_e *SlackApp_Expecter) AddSlashCommand(_a0 interface{}, _a1 interface{}) *SlackApp_AddSlashCommand_Call {
	return &SlackApp_AddSlashCommand_Call{Call: _e.mock.On("AddSlashCommand", _a0, _a1)}
}

func (_c *SlackApp_AddSlashCommand_Call) Run(run func(_a0 string, _a1 func(slack.SlashCommand, *socketmode.Client))) *SlackApp_AddSlashCommand_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(func(slack.SlashCommand, *socketmode.Client)))
	})
	return _c
}

func (_c *SlackApp_AddSlashCommand_Call) Return() *SlackApp_AddSlashCommand_Call {
	_c.Call.Return()
	return _c
}

func (_c *SlackApp_AddSlashCommand_Call) RunAndReturn(run func(string, func(slack.SlashCommand, *socketmode.Client))) *SlackApp_AddSlashCommand_Call {
	_c.Call.Return(run)
	return _c
}

// Run provides a mock function with given fields: ctx
func (_m *SlackApp) Run(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Run")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SlackApp_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type SlackApp_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - ctx context.Context
func (_e *SlackApp_Expecter) Run(ctx interface{}) *SlackApp_Run_Call {
	return &SlackApp_Run_Call{Call: _e.mock.On("Run", ctx)}
}

func (_c *SlackApp_Run_Call) Run(run func(ctx context.Context)) *SlackApp_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *SlackApp_Run_Call) Return(_a0 error) *SlackApp_Run_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *SlackApp_Run_Call) RunAndReturn(run func(context.Context) error) *SlackApp_Run_Call {
	_c.Call.Return(run)
	return _c
}

// NewSlackApp creates a new instance of SlackApp. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSlackApp(t interface {
	mock.TestingT
	Cleanup(func())
}) *SlackApp {
	mock := &SlackApp{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
