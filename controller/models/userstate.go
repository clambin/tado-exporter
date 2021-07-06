package models

type UserState int

const (
	UserHome UserState = iota
	UserAway UserState = iota
)
