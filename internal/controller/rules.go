package controller

import (
	"context"
	"embed"
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/tado-exporter/internal/poller"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Devices []Device
type Device struct {
	Name string
	Home bool
}

type Action interface {
	GetState() string
	GetDelay() time.Duration
	GetReason() string
	Description(includeDelay bool) string
	Do(context.Context, TadoClient) error
	slog.LogValuer
}

type Evaluator interface {
	Evaluate(Update) (Action, error)
}

type GroupEvaluator interface {
	Evaluator
	ParseUpdate(poller.Update) (Action, error)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// pushDevices pushes a slice of devices onto Lua's stack as a Table.
func pushDevices(l *lua.State, devices Devices) {
	l.NewTable()
	for i, p := range devices {
		l.NewTable()
		l.PushString(p.Name) // Push the value for "Name"
		l.SetField(-2, "Name")
		l.PushBoolean(p.Home) // Push the value for "Age"
		l.SetField(-2, "Home")
		l.RawSetInt(-2, i+1)
	}
}

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
