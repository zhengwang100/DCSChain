package ptypes

type StateType uint8

const (
	// normal case
	NEW_VIEW StateType = iota
	PREPREPARE
	PREPARE
	COMMIT
	REPLY

	// view-change
	VIEW_CHANGE
	CHECKPOINT
	VC_PREPARE
	VC_COMMIT
	VC_REPLY

	WAITING
)

// String: convert state type to string
func (st StateType) String() string {
	switch st {
	case 0:
		return "NEW_VIEW"
	case 1:
		return "PREPREPARE"
	case 2:
		return "PREPARE"
	case 3:
		return "COMMIT"
	case 4:
		return "REPLY"
	case 5:
		return "VIEW_CHANGE"
	case 6:
		return "CHECKPOINT"
	case 7:
		return "VC_PREPARE"
	case 8:
		return "VC_COMMIT"
	case 9:
		return "VC_REPLY"
	case 10:
		return "WAITING"
	default:
		return ""
	}
}

const CHECKPOINTNUM = 10
