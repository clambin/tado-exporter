package luart

import (
	"github.com/Shopify/go-lua"
	"io"
	"time"
)

func Compile(name string, r io.Reader) (*lua.State, error) {
	l := lua.NewState()
	lua.OpenLibraries(l)
	Register(l, time.Now)
	err := l.Load(r, name, "t")
	if err == nil {
		l.Call(0, 0)
	}
	return l, err
}

func Register(l *lua.State, now func() time.Time) {
	l.Register("IsInRange", isInRangeWithNow(now))
	l.Register("SecondsTill", secondsTillWithNow(now))
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
			// Range does not cross midnight
			inRange = currentMinutes >= startMinutes && currentMinutes <= endMinutes
		} else {
			// Range crosses midnight
			inRange = currentMinutes >= startMinutes || currentMinutes <= endMinutes
		}
		l.PushBoolean(inRange)
		return 1
	}
}

func secondsTillWithNow(now func() time.Time) lua.Function {
	if now == nil {
		now = time.Now
	}
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
