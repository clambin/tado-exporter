package controller

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"io"
)

type luaScript struct {
	*lua.State
}

func newLuaScript(name string, r io.Reader) (luaScript, error) {
	var script luaScript
	var err error
	script.State, err = luart.Compile(name, r)
	return script, err
}

func (r luaScript) initEvaluation() error {
	const evalName = "Evaluate"
	r.Global(evalName)
	if r.IsNil(-1) {
		return fmt.Errorf("lua does not contain %s function", evalName)
	}
	return nil
}
