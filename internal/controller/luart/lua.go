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
		startHour, _ := l.ToNumber(-4)
		startMinute, _ := l.ToNumber(-3)
		endHour, _ := l.ToNumber(-2)
		endMinute, _ := l.ToNumber(-1)

		yyyy, mon, dd := time.Now().Date()
		start := time.Date(yyyy, mon, dd, int(startHour), int(startMinute), 0, 0, time.Local)
		end := time.Date(yyyy, mon, dd, int(endHour), int(endMinute), 0, 0, time.Local)
		if end.Before(start) {
			end = end.Add(24 * time.Hour)
		}

		currentTime := now()
		inRange := !(currentTime.Before(start) || currentTime.After(end))
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
		yyy, mon, dd := time.Now().Date()
		to := time.Date(yyy, mon, dd, int(toHour), int(toMinute), 0, 0, time.Local)
		current := now()
		if to.Before(current) {
			to = to.Add(24 * time.Hour)
		}
		l.PushInteger(int((to.Sub(current)).Seconds()))
		return 1
	}
}
