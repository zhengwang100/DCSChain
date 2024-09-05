package common

// consensus protocol type
type ConsensusType string

const (
	HOTSTUFF_PROTOCOL_BASIC   ConsensusType = "basic"
	HOTSTUFF_PROTOCOL_CHAINED ConsensusType = "chained"
	HOTSTUFF_2_PROTOCOL       ConsensusType = "hotstuff2"
	PBFT                      ConsensusType = "pbft"
)
