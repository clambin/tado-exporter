// Code generated by mockery v2.15.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	slack "github.com/slack-go/slack"

	slackbot "github.com/clambin/go-common/slackbot"
)

// SlackBot is an autogenerated mock type for the SlackBot type
type SlackBot struct {
	mock.Mock
}

// Register provides a mock function with given fields: name, command
func (_m *SlackBot) Register(name string, command slackbot.CommandFunc) {
	_m.Called(name, command)
}

// Run provides a mock function with given fields: ctx
func (_m *SlackBot) Run(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Send provides a mock function with given fields: channel, attachments
func (_m *SlackBot) Send(channel string, attachments []slack.Attachment) error {
	ret := _m.Called(channel, attachments)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []slack.Attachment) error); ok {
		r0 = rf(channel, attachments)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewSlackBot interface {
	mock.TestingT
	Cleanup(func())
}

// NewSlackBot creates a new instance of SlackBot. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSlackBot(t mockConstructorTestingTNewSlackBot) *SlackBot {
	mock := &SlackBot{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}