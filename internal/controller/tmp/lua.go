package tmp

import (
	"embed"
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/tado-exporter/internal/controller/luart"
	"io"
	"os"
	"strings"
)

// loadLuaScript opens a Lua script from disk, an embedded file system, or as text.
func loadLuaScript(cfg ScriptConfig, fs embed.FS) (io.ReadCloser, error) {
	switch {
	case cfg.Text != "":
		return io.NopCloser(strings.NewReader(cfg.Text)), nil
	case cfg.Packaged != "":
		return fs.Open(cfg.Packaged)
	case cfg.Path != "":
		return os.Open(cfg.Path)
	default:
		return nil, fmt.Errorf("script config is empty")
	}
}

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
