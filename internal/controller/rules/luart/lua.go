package luart

import (
	"github.com/Shopify/go-lua"
	"io"
	"time"
)

func New() *lua.State {
	return NewWithTime(time.Now)
}

func NewWithTime(now func() time.Time) *lua.State {
	l := lua.NewState()
	lua.OpenLibraries(l)
	LoadTadoModule(now)(l)
	return l
}

func Compile(name string, r io.Reader) (*lua.State, error) {
	l := New()
	err := l.Load(r, name, "t")
	if err == nil {
		l.Call(0, 0)
	}
	return l, err
}
