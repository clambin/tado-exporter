package model

import "time"

type ZoneConfig struct {
	AutoAway     ZoneAutoAwayConfig
	LimitOverlay ZoneLimitOverlayConfig
	NightTime    ZoneNightTimeConfig
}

type ZoneAutoAwayConfig struct {
	Enabled bool
	Users   []int
	Delay   time.Duration
}

type ZoneLimitOverlayConfig struct {
	Enabled bool
	Delay   time.Duration
}

type ZoneNightTimeConfig struct {
	Enabled bool
	Time    ZoneNightTimestamp
}

type ZoneNightTimestamp struct {
	Hour    int
	Minutes int
}
