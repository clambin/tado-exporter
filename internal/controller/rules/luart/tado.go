package luart

import (
	"github.com/Shopify/go-lua"
	"time"
)

func LoadTadoModule(now func() time.Time) func(*lua.State) int {
	if now == nil {
		now = time.Now
	}
	functionsWithTime := map[string]func(func() time.Time) lua.Function{
		"isInRange":   isInRangeWithNow,
		"secondsTill": secondsTillWithNow,
	}
	return func(l *lua.State) int {
		l.NewTable()
		for name, f := range functionsWithTime {
			l.PushString(name)
			l.PushGoFunction(f(now))
			l.RawSet(-3)
		}
		l.SetGlobal("tado")
		return 1
	}
}

func isInRangeWithNow(now func() time.Time) lua.Function {
	return func(l *lua.State) int {
		startHour, _ := l.ToInteger(-4)
		startMinute, _ := l.ToInteger(-3)
		endHour, _ := l.ToInteger(-2)
		endMinute, _ := l.ToInteger(-1)

		// Get the time's hour and minute
		current := now().Local()
		hour, minute := current.Hour(), current.Minute()

		// Convert hours and minutes to "minutes since midnight" for easy comparison
		currentMinutes := hour*60 + minute
		startMinutes := startHour*60 + startMinute
		endMinutes := endHour*60 + endMinute

		var inRange bool
		if startMinutes <= endMinutes {
			// Range does not cross midnight: we need to be between start & end
			inRange = currentMinutes >= startMinutes && currentMinutes <= endMinutes
		} else {
			// Range crosses midnight: we need to be after current, or before end
			inRange = currentMinutes >= startMinutes || currentMinutes <= endMinutes
		}
		l.PushBoolean(inRange)
		return 1
	}
}

func secondsTillWithNow(now func() time.Time) lua.Function {
	return func(l *lua.State) int {
		toHour, _ := l.ToNumber(-2)
		toMinute, _ := l.ToNumber(-1)
		currentTime := now().Local()
		yyy, mon, dd := currentTime.Date()
		to := time.Date(yyy, mon, dd, int(toHour), int(toMinute), 0, 0, time.Local)
		if to.Before(currentTime) {
			to = to.Add(24 * time.Hour)
		}
		l.PushInteger(int((to.Sub(currentTime)).Seconds()))
		return 1
	}
}
