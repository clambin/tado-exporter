package model

import "time"

type ZoneConfig struct {
	Users        []int
	LimitOverlay ZoneLimitOverlayConfig
	NightTime    ZoneNightTimeConfig
}

type ZoneLimitOverlayConfig struct {
	Enabled bool
	Limit   time.Duration
}

type ZoneNightTimeConfig struct {
	Enabled bool
	Time    ZoneNightTimestamp
}

type ZoneNightTimestamp struct {
	Hour    int
	Minutes int
}
