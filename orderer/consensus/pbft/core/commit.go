package core

import (
	"encoding/json"
	ptypes "pbft/types"
)

// HandleCommitMsg: the node handle the commit message and update the checkpoint
func (p *PBFT) HandleCommitMsg(msg *ptypes.PMsg) *ptypes.PMsg {

	if p.CurPhase == ptypes.VIEW_CHANGE {
		return nil
	}

	// check this node receive enough commit message
	if !p.commited(msg) {
		return nil
	}

	for p.View.ViewNumber < msg.ViewNumber {
		p.View.NextView()
	}

	// stop the prepare timer
	p.PTimer.Timer.Stop()
	p.CurPhase = ptypes.COMMIT

	// generate the block validation from commit message
	validation := make([]byte, 0)
	valJson, err := json.Marshal(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs)
	if err == nil {
		validation = append(validation, valJson...)
	}
	p.BlkStore.CurProposalBlk.BlkHdr.Validation = validation

	// generate reply message and add it to local log
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
	// time.Sleep(20 * time.Millisecond)

	return replyMsg
}

// commited: check the node whether finished prepare phase and recieve n-f prepare message and 2f+1 commit message
// commited implement PBFT description as follow:
// We define the committed and committed-local predi-cates as follows: committed(m, v, n) is true if and only
// if prepared(m, v, n, i) is true for all i in some set of f+1 non-faulty replicas;
// and committed-local(m, v, n, i) is true ifand only if prepared is true and has accepted 2f+1 commits (possibly including its own)
// from different replicas that match the pre-prepare for m;
// a commit matches a pre-prepare if they have the same view, sequence number, and digest.
func (p *PBFT) commited(msg *ptypes.PMsg) bool {

	// check message matching view and signature
	// if ture, add it to message log
	if !p.CheckMsg(msg) {
		// fmt.Println(p.ConsId, "commited CheckMsg error", msg.ViewNumber, msg.SendNode)
		return false
	}

	// log the message
	p.LogMsg(msg)

	// check current phase
	if p.CurPhase == ptypes.COMMIT && msg.ViewNumber < p.View.ViewNumber {
		// fmt.Println(p.ConsId, "commited Phase error", msg.ViewNumber, msg.SendNode, p.CurPhase)
		return false
	}

	// the if clause replaces prepared()
	if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs) <= (p.View.NodesNum-1)/3*2 {
		return false
	}

	if len(p.MsgLog[p.View.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs) <= (p.View.NodesNum-1)/3*2 {
		return false
	}
	return true
}

// HandleNewPrepareMsg: the replica redo the protocol for messages
func (p *PBFT) HandleVCCommitMsg(msg *ptypes.PMsg) *ptypes.PMsg {

	// current phase must be view-change phase
	if p.CurPhase != ptypes.VIEW_CHANGE {
		return nil
	}
	// check this node receive enough prepare message
	if msg.ViewNumber != p.View.ViewNumber {
		return nil
	}

	// check the signature and log it
	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(1)) {
		return nil
	}
	p.LogMsg(msg)

	// check the threshold
	if len(p.ViewChangeMsgs.CommitMsgs) != (p.View.NodesNum-1)/3*2+1 {
		return nil
	}

	// stop timer because recieved enough VC-COMMIT message
	p.PTimer.Timer.Stop()

	// generate vc-reply message
	vcReplyMsg := &ptypes.PMsg{
		MType:      ptypes.VC_REPLY,
		ViewNumber: p.View.ViewNumber,
		SendNode:   p.GetNodeName(),
		OSet:       make([]*ptypes.PMsg, 0),
	}

	// get the reply seqence and avoid replying to messages you've already replied to
	// and if message's digest is null, also don't reply
	// each redo reply will be added into vc-reply OSet
	replySeq := p.GetReplySeqs()
	for _, m := range msg.OSet {
		if !p.Signer.VerifySign(m.SendNode, m.Signature, m.Message2Byte(1)) {
			return nil
		}

		// if the message has already been replied to or the summary of the message is null, it is not replied
		if IsContain(replySeq, m.SeqNum) || m.Digest == nil {
			continue
		}

		// generate the reply message and add to vc-reply message
		replyMsg := &ptypes.PMsg{
			MType:      ptypes.REPLY,
			ViewNumber: p.View.ViewNumber,
			SeqNum:     m.SeqNum,
			SendNode:   p.GetNodeName(),
			Digest:     m.Digest,
		}
		vcReplyMsg.OSet = append(vcReplyMsg.OSet, replyMsg)
	}

	// update local phase
	p.CurPhase = ptypes.NEW_VIEW
	p.View.NextView()
	p.SequenceNum = p.GetMaxSeqInOSets() + 1

	// log
	// p.Logger.Println("[VC-REPLY]: r_"+strconv.Itoa(p.ConsId), "View:", p.View.ViewNumber-1, "Seq:", p.SequenceNum)

	p.PTimer.Timer.Start(func() {
		p.Logger.Println("[TIMER-EXPIRE-VCREPLY]:", p.GetNodeName(), "View:", p.View.ViewNumber)
		p.StartViewChange()
	}, func() {
		// fmt.Println("vc timer stop")
	})

	return vcReplyMsg
}

// GetReplySeqs: get the slice of sequence of all local log reply messages
func (p *PBFT) GetReplySeqs() []int {
	if len(p.ReplyMsgs) == 0 {
		return []int{}
	}
	res := make([]int, 0)
	for _, replyMsg := range p.ReplyMsgs {
		res = append(res, replyMsg.SeqNum)
	}
	return res
}

// IsContain: check whether slice contains value
func IsContain(slice []int, value int) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// GenCheckPoint: execute the process of generating a checkpoint
func (p *PBFT) GenCheckPoint(msg *ptypes.PMsg) bool {
	// check whether execute the check point
	if (msg.SeqNum+1)%ptypes.CHECKPOINTNUM == 0 {

		// generate a checkpoint message and store it to local log
		checkpointMsg := &ptypes.PMsg{
			MType:    ptypes.CHECKPOINT,
			SeqNum:   p.SequenceNum,
			Digest:   msg.Digest,
			SendNode: p.GetNodeName(),
			ReciNode: "Broadcast",
		}
		sign := p.Signer.Sign(checkpointMsg.Message2Byte(1))
		checkpointMsg.Signature = sign

		// add the message to local checkpoint message log
		if _, ok := p.CheckPoint.CPMsgsBuffer[p.SequenceNum]; !ok {
			p.CheckPoint.CPMsgsBuffer[p.SequenceNum] = []*ptypes.PMsg{checkpointMsg}
		} else {
			p.CheckPoint.CPMsgsBuffer[p.SequenceNum] = append(p.CheckPoint.CPMsgsBuffer[p.SequenceNum], checkpointMsg)
		}

		// update the waiting checkpoint sequence for certification
		p.CheckPoint.WaitSeq = p.SequenceNum

		return true
	}
	return false
}

// GetMaxSeqInOsets: get the max sequence from the OSet in recieved message
func (p *PBFT) GetMaxSeqInOSets() int {
	maxSeq := 0
	for _, commitMsg := range p.ViewChangeMsgs.CommitMsgs {
		for _, oMsg := range commitMsg.OSet {
			if oMsg.SeqNum > maxSeq {
				maxSeq = oMsg.SeqNum
			}
		}
	}
	return maxSeq
}

// NewRound: start a new round consensus, refresh the consensus state
func (p *PBFT) NewRound() {
	p.View.NextView()
	p.CurProposal = ptypes.Proposal{}
	p.SequenceNum += 1
}
