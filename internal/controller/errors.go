package controller

import "errors"

var _ error = &errLuaInvalidResponse{}

type errLuaInvalidResponse struct {
	err error
}

func (e *errLuaInvalidResponse) Error() string {
	return "invalid lua response: " + e.err.Error()
}

func (e *errLuaInvalidResponse) Is(err error) bool {
	var errLuaInvalidResponse *errLuaInvalidResponse
	return errors.As(err, &errLuaInvalidResponse)
}
