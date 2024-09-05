package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	ptypes "pbft/types"
)

// HandlePrePrepareMsg: the node handle pre-prepare message
// HandlePrePrepareMsg implement PBFT description as follow:
// A backup accepts a pre-prepare message provided:...
// If backup i accepts the <<PRE-PREPARE, v, n, d>_p, m> message, it enters the prepare phase by multicasting a
// <PREPARE, v, n, d>_i message to all other replicas and adds both messages to its log. Otherwise, it does nothing.
// A replica (including the primary) accepts prepare messages and adds them to its log provided their
// signatures are correct, their view number equals the replica’s current view, and their sequence number is between h and H.
func (p *PBFT) HandlePrePrepareMsg(msg *ptypes.PMsg) *ptypes.PMsg {
	// fmt.Println(p.GetNodeName(), p.View.ViewNumber, msg.SendNode, msg.ViewNumber, msg.MType)
	// check the local phase
	// if p.CurPhase != ptypes.NEW_VIEW && p.CurPhase != ptypes.COMMIT {
	// 	return nil
	// }
	if p.MsgLog[msg.ViewNumber%ptypes.CHECKPOINTNUM].PreprepareMsg != nil {
		return nil
	}

	// check whether the message matching pre-prepare message
	if !p.CheckPrePrepareMsg(msg) {
		return nil
	}

	// the replica update local state for this view and stop the Timer which was set on last commit or view-change phase
	if !p.IsLeader() {
		p.UpdateConsensus(*msg)

		p.CurPhase = ptypes.PREPREPARE
		p.CurProposal = msg.Proposal
		p.SequenceNum = msg.SeqNum
		p.PTimer.Timer.Stop()
	}

	// check whether has recieved enough matching prepare messages or commit messages before the pre-prepare message
	waitingMsg := p.CheckPrepareAndCommit(msg)
	if waitingMsg != nil {
		// if the waiting is not nil
		// that says the node has recieved enough prepare messages or commit messages
		return waitingMsg
	}

	// generate the prepare message and sign for it
	prepareMsg := &ptypes.PMsg{
		MType:      ptypes.PREPARE,
		ViewNumber: p.View.ViewNumber,
		SeqNum:     msg.SeqNum,
		SendNode:   p.GetNodeName(),
		Digest:     msg.Digest,
		ReciNode:   "Broadcast",
	}
	sign := p.Signer.Sign(prepareMsg.Message2Byte(1))
	prepareMsg.Signature = sign

	// log
	// p.Logger.Println("[PREPARE]:", p.GetNodeName(), "View:", p.View.ViewNumber)

	return prepareMsg

}

// CheckPrePrepareMsg: a backup accepts a pre-prepare message provided
// CheckPrePrepareMsg implement PBFT description as follow:
// 1. the signatures in the request and the pre-prepare message are correct and d is the digest for m;
// 2. it is in view v;
// 3. it has not accepted a pre-prepare message for view v and sequence number containing a different digest;
// 4. the sequence number in the pre-prepare message is between a low water mark, h, and a high water mark, H.
func (p *PBFT) CheckPrePrepareMsg(msg *ptypes.PMsg) bool {

	if msg.ViewNumber != p.View.ViewNumber {
		fmt.Println("msg.ViewNumber != p.ViewNumber", p.ConsId, msg.ViewNumber, p.View.ViewNumber)
		return false
	}

	if p.SequenceNum > msg.SeqNum {
		fmt.Println("p.SequenceNum > msg.SeqNum", p.SequenceNum, msg.SeqNum)
		return false
	}

	if p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PreprepareMsg != nil &&
		p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PreprepareMsg.SeqNum == msg.SeqNum &&
		!bytes.Equal(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PreprepareMsg.Digest, msg.Digest) {
		fmt.Println("CheckPrePrepareMsg3", p.SequenceNum, msg.SeqNum)
		return false
	}

	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(0)) {
		fmt.Println("CheckPrePrepareMsg4", p.SequenceNum, msg.SeqNum, msg.SendNode, p.Signer.Pks[msg.SendNode])
		return false
	}

	p.LogMsg(msg)
	return true
}

// CheckPrepareAndCommit: a backup accepts a pre-prepare message provided
// params:
// - msg: the recieved pre-prepare message
// returns:
// - nil, the reply message or commit message
func (p *PBFT) CheckPrepareAndCommit(msg *ptypes.PMsg) *ptypes.PMsg {

	if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs) != 0 {
		commitMsgs := p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs
		p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs = make([]*ptypes.PMsg, 0, len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs))
		for _, commitMsg := range commitMsgs {
			if msg.ViewNumber == commitMsg.ViewNumber && msg.SeqNum == commitMsg.SeqNum && bytes.Equal(msg.Digest, commitMsg.Digest) {
				p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs = append(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs, commitMsg)
			}
		}

		if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs) <= (p.View.NodesNum-1)/3*2 {
			return nil
		}

		for p.View.ViewNumber < msg.ViewNumber {
			p.View.NextView()
		}

		// stop the prepare timer
		p.PTimer.Timer.Stop()
		p.CurPhase = ptypes.COMMIT

		validation := make([]byte, 0)
		valJson, err := json.Marshal(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs)
		if err == nil {
			validation = append(validation, valJson...)
		}
		p.BlkStore.CurProposalBlk.BlkHdr.Validation = validation

		replyMsg := &ptypes.PMsg{
			MType:      ptypes.REPLY,
			ViewNumber: p.View.ViewNumber,
			SeqNum:     p.SequenceNum,
			Digest:     msg.Digest,
			ReciNode:   msg.SendNode,
		}

		// log the reply message
		p.ReplyMsgs = append(p.ReplyMsgs, replyMsg)

		// store block and update with an empty block
		p.BlkStore.StoreBlock(p.BlkStore.CurProposalBlk)

		// check whether execute the check point

		if p.GenCheckPoint(replyMsg) {
			p.CurPhase = ptypes.CHECKPOINT
		} else {
			p.CurPhase = ptypes.NEW_VIEW
		}

		// update local state containing current phase, proposal, view number, sequence
		p.NewRound()

		// set a timer for liveness and ensure that the pre-prepare message from the next view leader is received within the specified time
		p.PTimer.Timer.Start(func() {
			p.Logger.Println("[TIMER-EXPIRE-REPLY]:", p.GetNodeName(), "View:", p.View.ViewNumber)
			p.StartViewChange()
		}, func() {
			// fmt.Println("prepare timer stop", p.GetNodeName(), p.View.ViewNumber)
		})

		// log
		// p.Logger.Println("[REPLY]:", p.GetNodeName(), "View:", p.View.ViewNumber-1, "Seq:", p.SequenceNum-1)
		return replyMsg
	} else if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs) != 0 {

		// check prepare messages
		prepareMsgs := p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs
		p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs = make([]*ptypes.PMsg, 0, len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs))
		for _, prepareMsg := range prepareMsgs {
			if msg.ViewNumber == prepareMsg.ViewNumber && msg.SeqNum == prepareMsg.SeqNum && bytes.Equal(msg.Digest, prepareMsg.Digest) {
				p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs = append(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs, prepareMsg)
			}
		}

		if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs) <= (p.View.NodesNum-1)/3*2 {
			return nil
		}

		// update local state
		for p.View.ViewNumber < msg.ViewNumber {
			p.View.NextView()
		}

		p.CurPhase = ptypes.PREPARE

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

		return commitMsg
	} else {
		return nil
	}
}
