package luart

import (
	"errors"
	"fmt"
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

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type LuaObject map[string]any

func GetObject(l *lua.State, index int) LuaObject {
	values := make(LuaObject)

	// Push nil to start the iteration
	l.PushNil()

	// Iterate over the table at the specified index
	for l.Next(index) {
		// Get the key as a string
		key, _ := l.ToString(-2)

		// Check the value type
		switch {
		case l.IsNumber(-1): // Number value
			v, _ := l.ToInteger(-1)
			values[key] = v
		case l.IsBoolean(-1): // Number value
			v := l.ToBoolean(-1)
			values[key] = v
		case l.IsString(-1): // String value
			v, _ := l.ToString(-1)
			values[key] = v
		case l.IsTable(-1):
			values[key] = GetObject(l, l.AbsIndex(-1))
		default:
			v := l.ToValue(-1)
			values[key] = v
		}
		// Pop the value
		l.Pop(1)
	}

	return values
}

func (o LuaObject) Push(l *lua.State) {
	l.NewTable() // Create a new table on the Lua stack

	for key, value := range o {
		// Push the key
		l.PushString(key)

		// Push the value based on its type
		switch v := value.(type) {
		case string:
			l.PushString(v)
		case float64: // Lua treats numbers as float64
			l.PushNumber(v)
		case int: // Convert int to float64 for Lua compatibility
			l.PushNumber(float64(v))
		case bool:
			l.PushBoolean(v)
		case map[string]any: // Recursively push nested maps
			LuaObject(v).Push(l)
		default:
			l.PushNil() // Unsupported types become nil
		}

		// Set the key-value pair in the table
		l.SetTable(-3)
	}
}

func GetObjectAttribute[T any](obj LuaObject, name string) (T, error) {
	var v T
	attrib, ok := obj[name]
	if !ok {
		return v, errors.New("not found")
	}
	if v, ok = attrib.(T); !ok {
		return v, fmt.Errorf("invalid type: %T", attrib)
	}
	return v, nil
}
