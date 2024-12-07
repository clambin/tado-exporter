package luart

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestTableToSlice(t *testing.T) {
	const returnValues = 3
	l, err := Compile(t.Name(), strings.NewReader(`
function foo()
	return true, { "a", 2, true }, true
end
`))
	require.NoError(t, err)

	l.Global("foo")
	require.False(t, l.IsNil(-1))
	require.NoError(t, l.ProtectedCall(0, returnValues, 0))

	values := make([]any, returnValues)
	values[0] = l.ToBoolean(-3)
	values[1] = TableToSlice(l, l.AbsIndex(-2))
	values[2] = l.ToBoolean(-1)
	l.Pop(returnValues)

	assert.True(t, l.IsNil(-1))
	assert.Equal(t, []any{true, []any{"a", 2, true}, true}, values)
}

func TestTableToMap(t *testing.T) {
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
	values[1] = TableToMap(l, l.AbsIndex(-2))
	values[2] = l.ToBoolean(-1)

	assert.Equal(t, []any{
		true,
		map[string]any{
			"Name": "foo",
			"Age":  24,
			"Address": map[string]any{
				"City": "Huldenberg",
			},
			"Home": true,
		},
		true,
	}, values)
}

func TestPushTable(t *testing.T) {
	const script = `
function foo(args) 
	return args.String, args.Int, args.Bool, args.Float
end
`

	args := map[string]any{
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
	PushMap(l, args)
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
