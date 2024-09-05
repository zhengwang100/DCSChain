package hs2types

// StateType: hostuff-2 protocol phase state, message type , QC type
type StateType uint8

const (
	NEW_VIEW StateType = iota
	NEW_PROPOSE
	PROPOSE
	VOTE1
	PREPARE
	VOTE2
	WISH
	TCMSG
	ENTER
)

// String: convert state type to string
func (st StateType) String() string {
	switch st {
	case 0:
		return "NEW_VIEW"
	case 1:
		return "NEW_PROPOSE"
	case 2:
		return "PROPOSE"
	case 3:
		return "VOTE1"
	case 4:
		return "PREPARE"
	case 5:
		return "VOTE2"
	case 6:
		return "WISH"
	case 7:
		return "TCMSG"
	case 8:
		return "ENTER"
	default:
		return ""
	}
}
