// Code generated by mockery v2.42.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	tado "github.com/clambin/tado"
)

// TadoGetter is an autogenerated mock type for the TadoGetter type
type TadoGetter struct {
	mock.Mock
}

type TadoGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *TadoGetter) EXPECT() *TadoGetter_Expecter {
	return &TadoGetter_Expecter{mock: &_m.Mock}
}

// GetHomeState provides a mock function with given fields: ctx
func (_m *TadoGetter) GetHomeState(ctx context.Context) (tado.HomeState, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetHomeState")
	}

	var r0 tado.HomeState
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (tado.HomeState, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) tado.HomeState); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(tado.HomeState)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TadoGetter_GetHomeState_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetHomeState'
type TadoGetter_GetHomeState_Call struct {
	*mock.Call
}

// GetHomeState is a helper method to define mock.On call
//   - ctx context.Context
func (_e *TadoGetter_Expecter) GetHomeState(ctx interface{}) *TadoGetter_GetHomeState_Call {
	return &TadoGetter_GetHomeState_Call{Call: _e.mock.On("GetHomeState", ctx)}
}

func (_c *TadoGetter_GetHomeState_Call) Run(run func(ctx context.Context)) *TadoGetter_GetHomeState_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *TadoGetter_GetHomeState_Call) Return(homeState tado.HomeState, err error) *TadoGetter_GetHomeState_Call {
	_c.Call.Return(homeState, err)
	return _c
}

func (_c *TadoGetter_GetHomeState_Call) RunAndReturn(run func(context.Context) (tado.HomeState, error)) *TadoGetter_GetHomeState_Call {
	_c.Call.Return(run)
	return _c
}

// GetMobileDevices provides a mock function with given fields: _a0
func (_m *TadoGetter) GetMobileDevices(_a0 context.Context) ([]tado.MobileDevice, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetMobileDevices")
	}

	var r0 []tado.MobileDevice
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]tado.MobileDevice, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []tado.MobileDevice); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]tado.MobileDevice)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TadoGetter_GetMobileDevices_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMobileDevices'
type TadoGetter_GetMobileDevices_Call struct {
	*mock.Call
}

// GetMobileDevices is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *TadoGetter_Expecter) GetMobileDevices(_a0 interface{}) *TadoGetter_GetMobileDevices_Call {
	return &TadoGetter_GetMobileDevices_Call{Call: _e.mock.On("GetMobileDevices", _a0)}
}

func (_c *TadoGetter_GetMobileDevices_Call) Run(run func(_a0 context.Context)) *TadoGetter_GetMobileDevices_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *TadoGetter_GetMobileDevices_Call) Return(_a0 []tado.MobileDevice, _a1 error) *TadoGetter_GetMobileDevices_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TadoGetter_GetMobileDevices_Call) RunAndReturn(run func(context.Context) ([]tado.MobileDevice, error)) *TadoGetter_GetMobileDevices_Call {
	_c.Call.Return(run)
	return _c
}

// GetWeatherInfo provides a mock function with given fields: _a0
func (_m *TadoGetter) GetWeatherInfo(_a0 context.Context) (tado.WeatherInfo, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetWeatherInfo")
	}

	var r0 tado.WeatherInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (tado.WeatherInfo, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) tado.WeatherInfo); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(tado.WeatherInfo)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TadoGetter_GetWeatherInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetWeatherInfo'
type TadoGetter_GetWeatherInfo_Call struct {
	*mock.Call
}

// GetWeatherInfo is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *TadoGetter_Expecter) GetWeatherInfo(_a0 interface{}) *TadoGetter_GetWeatherInfo_Call {
	return &TadoGetter_GetWeatherInfo_Call{Call: _e.mock.On("GetWeatherInfo", _a0)}
}

func (_c *TadoGetter_GetWeatherInfo_Call) Run(run func(_a0 context.Context)) *TadoGetter_GetWeatherInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *TadoGetter_GetWeatherInfo_Call) Return(_a0 tado.WeatherInfo, _a1 error) *TadoGetter_GetWeatherInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TadoGetter_GetWeatherInfo_Call) RunAndReturn(run func(context.Context) (tado.WeatherInfo, error)) *TadoGetter_GetWeatherInfo_Call {
	_c.Call.Return(run)
	return _c
}

// GetZoneInfo provides a mock function with given fields: _a0, _a1
func (_m *TadoGetter) GetZoneInfo(_a0 context.Context, _a1 int) (tado.ZoneInfo, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for GetZoneInfo")
	}

	var r0 tado.ZoneInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int) (tado.ZoneInfo, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int) tado.ZoneInfo); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(tado.ZoneInfo)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TadoGetter_GetZoneInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetZoneInfo'
type TadoGetter_GetZoneInfo_Call struct {
	*mock.Call
}

// GetZoneInfo is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 int
func (_e *TadoGetter_Expecter) GetZoneInfo(_a0 interface{}, _a1 interface{}) *TadoGetter_GetZoneInfo_Call {
	return &TadoGetter_GetZoneInfo_Call{Call: _e.mock.On("GetZoneInfo", _a0, _a1)}
}

func (_c *TadoGetter_GetZoneInfo_Call) Run(run func(_a0 context.Context, _a1 int)) *TadoGetter_GetZoneInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int))
	})
	return _c
}

func (_c *TadoGetter_GetZoneInfo_Call) Return(_a0 tado.ZoneInfo, _a1 error) *TadoGetter_GetZoneInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TadoGetter_GetZoneInfo_Call) RunAndReturn(run func(context.Context, int) (tado.ZoneInfo, error)) *TadoGetter_GetZoneInfo_Call {
	_c.Call.Return(run)
	return _c
}

// GetZones provides a mock function with given fields: _a0
func (_m *TadoGetter) GetZones(_a0 context.Context) (tado.Zones, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetZones")
	}

	var r0 tado.Zones
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (tado.Zones, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) tado.Zones); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(tado.Zones)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TadoGetter_GetZones_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetZones'
type TadoGetter_GetZones_Call struct {
	*mock.Call
}

// GetZones is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *TadoGetter_Expecter) GetZones(_a0 interface{}) *TadoGetter_GetZones_Call {
	return &TadoGetter_GetZones_Call{Call: _e.mock.On("GetZones", _a0)}
}

func (_c *TadoGetter_GetZones_Call) Run(run func(_a0 context.Context)) *TadoGetter_GetZones_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *TadoGetter_GetZones_Call) Return(_a0 tado.Zones, _a1 error) *TadoGetter_GetZones_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *TadoGetter_GetZones_Call) RunAndReturn(run func(context.Context) (tado.Zones, error)) *TadoGetter_GetZones_Call {
	_c.Call.Return(run)
	return _c
}

// NewTadoGetter creates a new instance of TadoGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewTadoGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *TadoGetter {
	mock := &TadoGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
