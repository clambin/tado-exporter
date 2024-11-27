package controller

import (
	"context"
	"embed"
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/clambin/go-common/set"
	"github.com/clambin/tado-exporter/internal/poller"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

type devices []device

func (d devices) filter(name set.Set[string]) devices {
	filtered := make(devices, 0, len(name))
	for _, entry := range d {
		if name.Contains(entry.Name) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

type device struct {
	Name string
	Home bool
}

type action interface {
	GetState() string
	GetDelay() time.Duration
	GetReason() string
	Description(includeDelay bool) string
	Do(context.Context, TadoClient) error
	slog.LogValuer
}

type evaluator interface {
	Evaluate(update) (action, error)
}

type groupEvaluator interface {
	evaluator
	ParseUpdate(poller.Update) (action, error)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// pushDevices pushes a slice of devices onto Lua's stack as a Table.
func pushDevices(l *lua.State, devices devices) {
	l.NewTable()
	for i, p := range devices {
		l.NewTable()
		l.PushString(p.Name) // Push the value for "Name"
		l.SetField(-2, "Name")
		l.PushBoolean(p.Home) // Push the value for "AtHome"
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
