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
	return true, { Name = "foo", Age = 24, Address = { City = "Huldenberg" } }, true
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
		},
		true,
	}, values)
}