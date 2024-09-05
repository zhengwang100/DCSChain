package core

import (
	ptypes "pbft/types"
)

// HandleCheckPoint: the node handles the check-point message
func (p *PBFT) HandleCheckPoint(msg *ptypes.PMsg) *ptypes.PMsg {

	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(1)) {
		return nil
	}

	// message's sequence < node's checkpoint sequence which indicates the message from past view and is out-dated
	if msg.SeqNum < p.CheckPoint.Seq {
		return nil
	}

	// if message's sequence is the same as check point sequence, the checkpoint garbage collection has finished!
	if msg.SeqNum == p.CheckPoint.Seq {
		return nil
	}

	// store message
	p.LogMsg(msg)

	// check threshold
	if len(p.CheckPoint.CPMsgsBuffer[msg.SeqNum]) <= (p.View.NodesNum-1)/3*2 {
		return nil
	}

	// update node local checkpoit included the sequence and certification which makes it a stable checkpoint
	p.CheckPoint.Seq = msg.SeqNum
	copy(p.CheckPoint.CPMsgs, p.CheckPoint.CPMsgsBuffer[msg.SeqNum])

	p.MsgLog = make([]ptypes.MsgsLog, ptypes.CHECKPOINTNUM)
	p.CheckPoint.CPMsgsBuffer[msg.SeqNum] = make([]*ptypes.PMsg, 0)

	// log
	// p.Logger.Println("[CHECK_POINT]:", p.GetNodeName(), "Succeed updating to", p.CheckPoint.Seq, "!")

	if p.IsLeader() {
		p.CurPhase = ptypes.WAITING
	} else {
		p.CurPhase = ptypes.NEW_VIEW
	}
	return nil
}
