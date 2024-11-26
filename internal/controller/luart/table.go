package luart

import "github.com/Shopify/go-lua"

func TableToSlice(l *lua.State, index int) []any {
	var values []any

	l.PushNil()
	// Iterate over the table at the specified index
	for l.Next(index) {
		// Check the value type
		switch {
		case l.IsNumber(-1): // Number value
			v, _ := l.ToInteger(-1)
			values = append(values, v)
		case l.IsBoolean(-1): // Number value
			v := l.ToBoolean(-1)
			values = append(values, v)
		case l.IsString(-1): // String value
			v, _ := l.ToString(-1)
			values = append(values, v)
		default:
			v := l.ToValue(-1)
			values = append(values, v)
		}
		// Pop the value
		l.Pop(1)
	}

	return values
}

func TableToMap(l *lua.State, index int) map[string]interface{} {
	values := make(map[string]interface{})

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
			values[key] = TableToMap(l, l.AbsIndex(-1))
		default:
			v := l.ToValue(-1)
			values[key] = v
		}
		// Pop the value
		l.Pop(1)
	}

	return values
}
