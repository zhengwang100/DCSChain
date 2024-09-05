package ptypes

// MsgLog: the log struct of recieved message in per view
// one view corresponds to one MsgLog
type MsgsLog struct {
	SelfMsgs      []*PMsg // the set of self-sent messages
	NewViewMsgs   []*PMsg // the set of recieved new-view messages
	PreprepareMsg *PMsg   // the recieved pre-prepare message
	PrepareMsgs   []*PMsg // the set of recieved prepare messages
	CommitMsgs    []*PMsg // the set of recieved commit messages
}

// IsEmpty: check whether MsgLog is empty
func (ml *MsgsLog) IsEmpty() bool {
	if ml == nil || ml.PreprepareMsg == nil {
		return true
	}
	return ml.PreprepareMsg.Proposal.IsEmpty() || len(ml.PrepareMsgs) == 0
}
