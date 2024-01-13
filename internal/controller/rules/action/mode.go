package action

type Mode int

func (m Mode) String() string {
	if m >= 0 && int(m) < len(modeNames) {
		return modeNames[m]
	}
	return "unknown"
}

const (
	NoAction Mode = iota
	HomeInHomeMode
	HomeInAwayMode
	ZoneInOverlayMode
	ZoneInAutoMode
)

var modeNames = []string{
	"no action",
	"home",
	"away",
	"overlay",
	"auto",
}
