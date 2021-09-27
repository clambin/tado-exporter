package processor

import "time"

type ZoneRules struct {
	AutoAway     ZoneAutoAwayRule
	LimitOverlay ZoneLimitOverlayRule
	NightTime    ZoneNightTimeRule
}

type ZoneAutoAwayRule struct {
	Enabled bool
	Users   []int
	Delay   time.Duration
}

type ZoneLimitOverlayRule struct {
	Enabled bool
	Delay   time.Duration
}

type ZoneNightTimeRule struct {
	Enabled bool
	Time    ZoneNightTimestamp
}

type ZoneNightTimestamp struct {
	Hour    int
	Minutes int
}
