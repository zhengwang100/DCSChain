package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"message"
	ptypes "pbft/types"
	"strconv"
)

// IsLeader: return whether the current node is a leader(true if leader)
func (p *PBFT) IsLeader() bool {
	return p.View.Leader == p.ConsId
}

// GetNodeName： get the node name(string) by converting PBFT core ID to name
func (p *PBFT) GetNodeName() string {
	return "r_" + strconv.Itoa(p.ConsId)
}

// CheckMsg: check message view, signature for safety, whether it matches the pre-prepare message in the corresponding view
func (p *PBFT) CheckMsg(msg *ptypes.PMsg) bool {

	// check view number and sequence number
	if msg.ViewNumber < p.View.ViewNumber || msg.SeqNum > p.CheckPoint.Seq+ptypes.CHECKPOINTNUM {
		// fmt.Println(p.ConsId, p.CurPhase, "ViewNumber check failed", msg.MType, msg.ViewNumber, p.View.ViewNumber)
		return false
	}

	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(1)) {
		fmt.Println(p.ConsId, "p.VerifySign failed")
		return false
	}

	if msg.MType != ptypes.PREPREPARE {
		if !p.MatchPrePrepareMsg(msg) {
			fmt.Println(p.ConsId, msg.MType, msg.SendNode, "match pre-prepare message failed")
			return false
		}
	}

	return true
}

// MatchPrePrepareMsg: check whether the message is matching the pre-prepare message in the same view
func (p *PBFT) MatchPrePrepareMsg(msg *ptypes.PMsg) bool {
	ppMsg := p.MsgLog[msg.ViewNumber%ptypes.CHECKPOINTNUM].PreprepareMsg
	if ppMsg == nil {
		fmt.Println("MatchPrePrepareMsg failed empty", msg.ViewNumber, p.View.ViewNumber, p.CheckPoint.Seq, msg.ViewNumber%ptypes.CHECKPOINTNUM)
		p.LogMsg(msg)
		return false
	}
	return msg.ViewNumber == ppMsg.ViewNumber && msg.SeqNum == ppMsg.SeqNum && bytes.Equal(msg.Digest, ppMsg.Digest)
}

// LogMsg: log the message by its type
func (p *PBFT) LogMsg(msg *ptypes.PMsg) {
	switch msg.MType {
	case ptypes.NEW_VIEW:
		if _, ok := p.NewViewMsgs[msg.ViewNumber]; !ok {
			// if no, add a key-value pair
			p.NewViewMsgs[msg.ViewNumber] = []*ptypes.PMsg{msg}
		} else {
			p.NewViewMsgs[msg.ViewNumber] = append(p.NewViewMsgs[msg.ViewNumber], msg)
		}
		if len(p.NewViewMsgs[msg.ViewNumber]) > (p.View.NodesNum-1)/3*2 && len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].NewViewMsgs) == 0 {
			// fmt.Println("logmsg", len(p.NewViewMsgs[msg.ViewNumber]))
			p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].NewViewMsgs = append(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].NewViewMsgs, p.NewViewMsgs[msg.ViewNumber]...)
			p.NewViewMsgs[msg.ViewNumber] = []*ptypes.PMsg{}
		}
	case ptypes.PREPREPARE:
		p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PreprepareMsg = msg
	case ptypes.PREPARE:
		p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs = append(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs, msg)
	case ptypes.COMMIT:
		p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs = append(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs, msg)
	case ptypes.CHECKPOINT:
		p.CheckPoint.CPMsgsBuffer[msg.SeqNum] = append(p.CheckPoint.CPMsgsBuffer[msg.SeqNum], msg)
	case ptypes.VIEW_CHANGE:
		p.ViewChangeMsgs.NewViewMsgs = append(p.ViewChangeMsgs.NewViewMsgs, msg)
	case ptypes.VC_PREPARE:
		p.ViewChangeMsgs.PrepareMsgs = append(p.ViewChangeMsgs.PrepareMsgs, msg)
	case ptypes.VC_COMMIT:
		p.ViewChangeMsgs.CommitMsgs = append(p.ViewChangeMsgs.CommitMsgs, msg)
	}
}

// UpdateNode: refresh the node in this system
func (p *PBFT) UpdateConsensus(msg ptypes.PMsg) {
	p.CurProposal = msg.Proposal
	p.BlkStore.CurProposalBlk = msg.Block
	p.BlkStore.CurBlkHash = p.BlkStore.CurProposalBlk.Hash()
}

// GetValidMsgs: get all valid pre-prepare message and matching prepare message
// valid: a pre-prepare message has 2f+1 matching prepare message which is considered to be valid
func (p *PBFT) GetValidMsgs() []*ptypes.Pm {
	pm := make([]*ptypes.Pm, 0)
	j := 0
	threshold := (p.View.NodesNum - 1) / 3 * 2
	for i := 0; i < len(p.MsgLog); i++ {
		if p.MsgLog[i].IsEmpty() || len(p.MsgLog[i].PrepareMsgs) <= threshold {
			break
		}

		pm = append(pm, &ptypes.Pm{
			PrePrepareMsg: p.MsgLog[i].PreprepareMsg.PMsg2VCMsg(),
			PrepareMsgs:   make([]*ptypes.VCMsg, 0),
		})
		for _, v := range p.MsgLog[i].PrepareMsgs {
			pm[j].PrepareMsgs = append(pm[j].PrepareMsgs, v.PMsg2VCMsg())
		}
		j++
	}
	return pm
}

// ReSetViewchangeMsgs: reset the veiw-change message log
func (p *PBFT) ReSetViewchangeMsgs() {
	p.ViewChangeMsgs = ptypes.MsgsLog{}
}

// SendSerMsg: send the message, in fact the chan provided by the outer layer is passed to the outer layer,
// and the outer layer sends the message
// params:
// - msg: message that need to be sent
func (p *PBFT) SendSerMsg(msg *ptypes.PMsg) {
	msgJson, err := json.Marshal(msg)
	if err != nil {
		return
	}
	serMsg := message.ServerMsg{
		SType:      message.ORDER,
		SendServer: msg.SendNode,
		ReciServer: msg.ReciNode,
		Payload:    msgJson,
	}

	p.SendChan <- serMsg
}

// initLeader: the node which is leader in view init
func (p *PBFT) InitLeader() {
	p.CurPhase = ptypes.WAITING
}

// GetLeaderName: get leader of current view name
func (p *PBFT) GetLeaderName() string {
	return p.View.LeaderName()
}

// Execute: execute the commands
func (p *PBFT) Execute() bool {
	return true
}
