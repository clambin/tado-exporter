package luart

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestGetObject(t *testing.T) {
	const returnValues = 3
	l, err := Compile(t.Name(), strings.NewReader(`
function foo()
	return true, { Name = "foo", Age = 24, Address = { City = "Huldenberg" }, Home = true }, true
end
`))
	require.NoError(t, err)

	l.Global("foo")
	require.False(t, l.IsNil(-1))
	require.NoError(t, l.ProtectedCall(0, returnValues, 0))

	values := make([]any, returnValues)
	values[0] = l.ToBoolean(-3)
	values[1] = GetObject(l, l.AbsIndex(-2))
	values[2] = l.ToBoolean(-1)

	o := LuaObject{
		"Name": "foo",
		"Age":  24,
		"Address": LuaObject{
			"City": "Huldenberg",
		},
		"Home": true,
	}
	assert.Equal(t, []any{true, o, true}, values)

	name, err := GetObjectAttribute[string](o, "Name")
	require.NoError(t, err)
	assert.Equal(t, "foo", name)
	age, err := GetObjectAttribute[int](o, "Age")
	require.NoError(t, err)
	assert.Equal(t, 24, age)
	home, err := GetObjectAttribute[bool](o, "Home")
	require.NoError(t, err)
	assert.True(t, home)

	_, err = GetObjectAttribute[string](o, "Missing")
	require.Error(t, err)
	_, err = GetObjectAttribute[int](o, "Name")
	require.Error(t, err)

}

func TestPushObject(t *testing.T) {
	const script = `
function foo(args) 
	return args.String, args.Int, args.Bool, args.Float
end
`

	args := LuaObject{
		"String": "foo",
		"Int":    10,
		"Bool":   true,
		"Float":  3.14159,
	}

	argCount := len(args)

	l, err := Compile(t.Name(), strings.NewReader(script))
	require.NoError(t, err)

	l.Global("foo")
	require.NotNil(t, l.IsNil(-1))
	args.Push(l)
	l.Call(1, argCount)

	for i, key := range []string{"String", "Int", "Bool", "Float"} {
		var value any
		var ok bool
		switch args[key].(type) {
		case string:
			value, ok = l.ToString(-argCount + i)
		case int:
			value, ok = l.ToInteger(-argCount + i)
		case bool:
			value = l.ToBoolean(-argCount + i)
			ok = true
		case float64:
			value, ok = l.ToNumber(-argCount + i)
		default:
			t.Fatal("unsupported type")
		}
		assert.True(t, ok)
		assert.Equal(t, value, args[key])
	}

	l.Pop(len(args))
}
