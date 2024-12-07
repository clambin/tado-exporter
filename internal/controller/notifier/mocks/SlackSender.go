// Code generated by mockery v2.50.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	slack "github.com/slack-go/slack"
)

// SlackSender is an autogenerated mock type for the SlackSender type
type SlackSender struct {
	mock.Mock
}

type SlackSender_Expecter struct {
	mock *mock.Mock
}

func (_m *SlackSender) EXPECT() *SlackSender_Expecter {
	return &SlackSender_Expecter{mock: &_m.Mock}
}

// AuthTest provides a mock function with no fields
func (_m *SlackSender) AuthTest() (*slack.AuthTestResponse, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for AuthTest")
	}

	var r0 *slack.AuthTestResponse
	var r1 error
	if rf, ok := ret.Get(0).(func() (*slack.AuthTestResponse, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *slack.AuthTestResponse); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack.AuthTestResponse)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SlackSender_AuthTest_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AuthTest'
type SlackSender_AuthTest_Call struct {
	*mock.Call
}

// AuthTest is a helper method to define mock.On call
func (_e *SlackSender_Expecter) AuthTest() *SlackSender_AuthTest_Call {
	return &SlackSender_AuthTest_Call{Call: _e.mock.On("AuthTest")}
}

func (_c *SlackSender_AuthTest_Call) Run(run func()) *SlackSender_AuthTest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *SlackSender_AuthTest_Call) Return(_a0 *slack.AuthTestResponse, _a1 error) *SlackSender_AuthTest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *SlackSender_AuthTest_Call) RunAndReturn(run func() (*slack.AuthTestResponse, error)) *SlackSender_AuthTest_Call {
	_c.Call.Return(run)
	return _c
}

// GetConversations provides a mock function with given fields: _a0
func (_m *SlackSender) GetConversations(_a0 *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetConversations")
	}

	var r0 []slack.Channel
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(*slack.GetConversationsParameters) ([]slack.Channel, string, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(*slack.GetConversationsParameters) []slack.Channel); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]slack.Channel)
		}
	}

	if rf, ok := ret.Get(1).(func(*slack.GetConversationsParameters) string); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(*slack.GetConversationsParameters) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// SlackSender_GetConversations_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetConversations'
type SlackSender_GetConversations_Call struct {
	*mock.Call
}

// GetConversations is a helper method to define mock.On call
//   - _a0 *slack.GetConversationsParameters
func (_e *SlackSender_Expecter) GetConversations(_a0 interface{}) *SlackSender_GetConversations_Call {
	return &SlackSender_GetConversations_Call{Call: _e.mock.On("GetConversations", _a0)}
}

func (_c *SlackSender_GetConversations_Call) Run(run func(_a0 *slack.GetConversationsParameters)) *SlackSender_GetConversations_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*slack.GetConversationsParameters))
	})
	return _c
}

func (_c *SlackSender_GetConversations_Call) Return(_a0 []slack.Channel, _a1 string, _a2 error) *SlackSender_GetConversations_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *SlackSender_GetConversations_Call) RunAndReturn(run func(*slack.GetConversationsParameters) ([]slack.Channel, string, error)) *SlackSender_GetConversations_Call {
	_c.Call.Return(run)
	return _c
}

// GetUsersInConversation provides a mock function with given fields: params
func (_m *SlackSender) GetUsersInConversation(params *slack.GetUsersInConversationParameters) ([]string, string, error) {
	ret := _m.Called(params)

	if len(ret) == 0 {
		panic("no return value specified for GetUsersInConversation")
	}

	var r0 []string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(*slack.GetUsersInConversationParameters) ([]string, string, error)); ok {
		return rf(params)
	}
	if rf, ok := ret.Get(0).(func(*slack.GetUsersInConversationParameters) []string); ok {
		r0 = rf(params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(*slack.GetUsersInConversationParameters) string); ok {
		r1 = rf(params)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(*slack.GetUsersInConversationParameters) error); ok {
		r2 = rf(params)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// SlackSender_GetUsersInConversation_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetUsersInConversation'
type SlackSender_GetUsersInConversation_Call struct {
	*mock.Call
}

// GetUsersInConversation is a helper method to define mock.On call
//   - params *slack.GetUsersInConversationParameters
func (_e *SlackSender_Expecter) GetUsersInConversation(params interface{}) *SlackSender_GetUsersInConversation_Call {
	return &SlackSender_GetUsersInConversation_Call{Call: _e.mock.On("GetUsersInConversation", params)}
}

func (_c *SlackSender_GetUsersInConversation_Call) Run(run func(params *slack.GetUsersInConversationParameters)) *SlackSender_GetUsersInConversation_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*slack.GetUsersInConversationParameters))
	})
	return _c
}

func (_c *SlackSender_GetUsersInConversation_Call) Return(_a0 []string, _a1 string, _a2 error) *SlackSender_GetUsersInConversation_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *SlackSender_GetUsersInConversation_Call) RunAndReturn(run func(*slack.GetUsersInConversationParameters) ([]string, string, error)) *SlackSender_GetUsersInConversation_Call {
	_c.Call.Return(run)
	return _c
}

// PostMessage provides a mock function with given fields: _a0, _a1
func (_m *SlackSender) PostMessage(_a0 string, _a1 ...slack.MsgOption) (string, string, error) {
	_va := make([]interface{}, len(_a1))
	for _i := range _a1 {
		_va[_i] = _a1[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _a0)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for PostMessage")
	}

	var r0 string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(string, ...slack.MsgOption) (string, string, error)); ok {
		return rf(_a0, _a1...)
	}
	if rf, ok := ret.Get(0).(func(string, ...slack.MsgOption) string); ok {
		r0 = rf(_a0, _a1...)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string, ...slack.MsgOption) string); ok {
		r1 = rf(_a0, _a1...)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(string, ...slack.MsgOption) error); ok {
		r2 = rf(_a0, _a1...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// SlackSender_PostMessage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PostMessage'
type SlackSender_PostMessage_Call struct {
	*mock.Call
}

// PostMessage is a helper method to define mock.On call
//   - _a0 string
//   - _a1 ...slack.MsgOption
func (_e *SlackSender_Expecter) PostMessage(_a0 interface{}, _a1 ...interface{}) *SlackSender_PostMessage_Call {
	return &SlackSender_PostMessage_Call{Call: _e.mock.On("PostMessage",
		append([]interface{}{_a0}, _a1...)...)}
}

func (_c *SlackSender_PostMessage_Call) Run(run func(_a0 string, _a1 ...slack.MsgOption)) *SlackSender_PostMessage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]slack.MsgOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(slack.MsgOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *SlackSender_PostMessage_Call) Return(_a0 string, _a1 string, _a2 error) *SlackSender_PostMessage_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *SlackSender_PostMessage_Call) RunAndReturn(run func(string, ...slack.MsgOption) (string, string, error)) *SlackSender_PostMessage_Call {
	_c.Call.Return(run)
	return _c
}

// NewSlackSender creates a new instance of SlackSender. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSlackSender(t interface {
	mock.TestingT
	Cleanup(func())
}) *SlackSender {
	mock := &SlackSender{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
