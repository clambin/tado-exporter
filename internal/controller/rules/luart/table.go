package luart

import "github.com/Shopify/go-lua"

func PushMap(l *lua.State, m map[string]any) {
	l.NewTable() // Create a new table on the Lua stack

	for key, value := range m {
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
			PushMap(l, v)
		default:
			l.PushNil() // Unsupported types become nil
		}

		// Set the key-value pair in the table
		l.SetTable(-3)
	}
}

func TableToMap(l *lua.State, index int) map[string]any {
	values := make(map[string]any)

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
		//case l.IsTable(-1):
		//	values = append(values, TableToSlice(l, l.AbsIndex(-1)))
		default:
			v := l.ToValue(-1)
			values = append(values, v)
		}
		// Pop the value
		l.Pop(1)
	}

	return values
}
