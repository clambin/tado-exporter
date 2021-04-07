package model

type UserState int

const (
	UserUnknown UserState = 0
	UserHome    UserState = 1
	UserAway    UserState = 2
)
