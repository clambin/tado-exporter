package rules

import "errors"

var _ error = &errLua{}

type errLua struct {
	err error
}

func (e *errLua) Error() string {
	return "lua: " + e.err.Error()
}

func (e *errLua) Is(err error) bool {
	var errLua *errLua
	return errors.As(err, &errLua)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var _ error = &errLuaInvalidResponse{}

type errLuaInvalidResponse struct {
	err error
}

func (e *errLuaInvalidResponse) Error() string {
	return "lua: invalid response: " + e.err.Error()
}

func (e *errLuaInvalidResponse) Is(err error) bool {
	var errLuaInvalidResponse *errLuaInvalidResponse
	return errors.As(err, &errLuaInvalidResponse)
}
