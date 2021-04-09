package model

type UserState int

const (
	UserUnknown UserState = 0
	UserHome    UserState = 1
	UserAway    UserState = 2
)

/*
func (state UserState) String() string {
	switch state {
	case UserHome:
		return "home"
	case UserAway:
		return "away"
	}
	return "unknown"
}
*/
