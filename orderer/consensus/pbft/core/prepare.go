package core

import (
	ptypes "pbft/types"
)

// HandlePrepareMsg: the node handle prepare messages
// HandlePrepareMsg implement PBFT description as follow:
// Replica multicasts a <COMMIT, v, n, d, i>_i to the other replicas when prepared(m, v, n, i) becomes true.
// This starts the commit phase. Replicas accept commit messages and insert them in their log provided they are properly signed,
// the view number in the message is equalto the replica’s current view, and the sequence number is between h and H.
func (p *PBFT) HandlePrepareMsg(msg *ptypes.PMsg) *ptypes.PMsg {

	// check the local phase
	if p.CurPhase == ptypes.VIEW_CHANGE {
		return nil
	}

	// check this node receive enough prepare message
	if !p.prepared(msg) {
		// fmt.Println(p.ConsId, "prepared CheckMsg error", msg.ViewNumber, msg.SendNode)
		return nil
	}

	// update local state
	p.CurPhase = ptypes.PREPARE

	for p.View.ViewNumber < msg.ViewNumber {
		p.View.NextView()
	}

	// generate the commit message and sign for it
	commitMsg := &ptypes.PMsg{
		MType:      ptypes.COMMIT,
		ViewNumber: p.View.ViewNumber,
		SeqNum:     p.SequenceNum,
		SendNode:   p.GetNodeName(),
		Digest:     msg.Digest,
		ReciNode:   "Broadcast",
	}
	sign := p.Signer.Sign(commitMsg.Message2Byte(1))
	commitMsg.Signature = sign

	// log
	// p.Logger.Println("[COMMIT]:", p.GetNodeName(), "View:", p.View.ViewNumber)

	return commitMsg
}

// prepared: check the node whether finished prepare phase and 2f+1 commit message
// prepared implement PBFT description as follow:
// We define the predicate prepared(m, v, n, i) to be true if and only if replica i has inserted in its log:
// the request m, a pre-prepare for m in view v with sequence number n, and 2 prepares from different backups that match
// the pre-prepare. The replicas verify whether the prepares match the pre-prepare by checking that they have the
// same view, sequence number, and digest.
func (p *PBFT) prepared(msg *ptypes.PMsg) bool {
	// check message whether is matching the view and pre-prepare message
	// if pass check, add it to message log
	if !p.CheckMsg(msg) {
		// fmt.Println(p.ConsId, "prepared CheckMsg error")
		return false
	}
	p.LogMsg(msg)

	// check local phase
	// if p.CurPhase != ptypes.PREPREPARE && p.CurPhase != ptypes.NEW_VIEW {
	if p.CurPhase == ptypes.PREPARE {
		// fmt.Println(p.ConsId, "prepared Phase error", p.CurPhase)
		return false
	}

	// check threshold
	if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs) <= (p.View.NodesNum-1)/3*2 {
		return false
	}
	return true
}

// HandleVCPrepareMsg: handle the vc-prepare message when the replica redo the protocol for messages
func (p *PBFT) HandleVCPrepareMsg(msg *ptypes.PMsg) *ptypes.PMsg {
	// check the local phase
	if p.CurPhase != ptypes.VIEW_CHANGE {
		return nil
	}

	if msg.ViewNumber != p.View.ViewNumber {
		return nil
	}

	// check the signature and log it
	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(1)) {
		return nil
	}
	p.LogMsg(msg)

	// check this node receive enough prepare message
	if len(p.ViewChangeMsgs.PrepareMsgs) != (p.View.NodesNum-1)/3*2+1 {
		return nil
	}

	// generate the vc-commit message and sign for it
	vcCommitMsg := &ptypes.PMsg{
		MType:      ptypes.VC_COMMIT,
		ViewNumber: p.View.ViewNumber,
		SendNode:   p.GetNodeName(),
		ReciNode:   "Broadcast",
		OSet:       make([]*ptypes.PMsg, 0),
	}

	// Each prepare message carried in the vc-prepare message is checked
	// and the corresponding commit message is generated and signed
	for _, m := range msg.OSet {
		if !p.Signer.VerifySign(m.SendNode, m.Signature, m.Message2Byte(1)) {
			return nil
		}
		commitMsg := &ptypes.PMsg{
			MType:      ptypes.COMMIT,
			ViewNumber: p.View.ViewNumber,
			SeqNum:     m.SeqNum,
			SendNode:   p.GetNodeName(),
			Digest:     m.Digest,
			ReciNode:   "Broadcast",
		}
		sign := p.Signer.Sign(commitMsg.Message2Byte(1))
		commitMsg.Signature = sign
		vcCommitMsg.OSet = append(vcCommitMsg.OSet, commitMsg)
	}

	// sign the vc-commit message and add it
	sign := p.Signer.Sign(vcCommitMsg.Message2Byte(1))
	vcCommitMsg.Signature = sign

	// log
	p.Logger.Println("[VC-COMMIT]:", p.GetNodeName(), "View:", p.View.ViewNumber)

	return vcCommitMsg
}
