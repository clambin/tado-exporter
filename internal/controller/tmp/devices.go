package tmp

import (
	"github.com/Shopify/go-lua"
	"github.com/clambin/go-common/set"
)

type devices []device

func (d devices) filter(names set.Set[string]) devices {
	emptyList := len(names) == 0
	filtered := make(devices, 0, len(names))
	for _, entry := range d {
		if emptyList || names.Contains(entry.Name) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (d devices) toLua(l *lua.State) {
	l.NewTable()
	for i, p := range d {
		l.NewTable()
		l.PushString(p.Name) // Push the value for "Name"
		l.SetField(-2, "Name")
		l.PushBoolean(p.Home) // Push the value for "AtHome"
		l.SetField(-2, "Home")
		l.RawSetInt(-2, i+1)
	}
}

type device struct {
	Name string
	Home bool
}
