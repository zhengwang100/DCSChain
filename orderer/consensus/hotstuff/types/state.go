package hstypes

// StateType: the consensus state which is during phase
type StateType uint8

const (
	// basic-hostuff
	NEW_VIEW StateType = iota
	PREPARE
	PREPARE_VOTE
	PRE_COMMIT
	PRE_COMMIT_VOTE
	COMMIT
	COMMIT_VOTE
	DECIDE

	// chained-hostuff
	CHINED_NEW_VIEW
	HALF_GENERIC
	GENERIC
	GENERIC_VOTE
	WAITING

	// node join or exit
	JOIN
	EXIT
	PREPARE_CERT
	COMMIT_CERT
	SYNC
	AGREE

	RESTART
)

// String: convert state type to string
func (st StateType) String() string {
	switch st {
	case 0:
		return "NEW_VIEW"
	case 1:
		return "PREPARE"
	case 2:
		return "PREPARE_VOTE"
	case 3:
		return "PRE_COMMIT"
	case 4:
		return "PRE_COMMIT_VOTE"
	case 5:
		return "COMMIT"
	case 6:
		return "COMMIT_VOTE"
	case 7:
		return "DECIDE"
	case 8:
		return "CHINED_NEW_VIEW"
	case 9:
		return "HALF_GENERIC"
	case 10:
		return "GENERIC"
	case 11:
		return "GENERIC_VOTE"
	case 12:
		return "WAITING"
	case 13:
		return "JOIN"
	case 14:
		return "EXIT"
	case 15:
		return "PREPARE_CERT"
	case 16:
		return "COMMIT_CERT"
	case 17:
		return "SYNC"
	case 18:
		return "AGREE"
	case 19:
		return "RESTART"
	default:
		return ""
	}
}
