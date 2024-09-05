package core

import (
	"common"
	"fmt"
	ptypes "pbft/types"
)

// HandleNewViewMsg: the replica handle new view message from leader of the view v+1
// HandleNewViewMsg implement PBFT description as follow:
// A backup accepts a new-view message for view v+1 if it is signed properly, if the view-change messages it
// contains are valid for view v+1, and if the set O is correct; it verifies the correctness of by performing a
// computation similar to the one used by the primary to create O. Then it adds the new information to its log as
// described for the primary, multicasts a prepare for each message in O to all the other replicas, adds these prepares
// to its log, and enters view v+1.
func (p *PBFT) HandleNewViewMsg(msg *ptypes.PMsg) *ptypes.PMsg {

	// verify signature and log it
	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(3)) {
		fmt.Println("HandleNewViewMsg VerifySign error")
		return nil
	}
	p.LogMsg(msg)

	if msg.ViewNumber != p.View.ViewNumber {
		return nil
	}

	// check the OSet of the message
	if !p.VerifyOSet(msg) {
		p.Logger.Println("VerifyOSet Err", len(msg.OSet), p.GetNodeName())
		return nil
	}

	// update local state and clear the previous state
	p.CurProposal = ptypes.Proposal{}
	p.BlkStore.GenEmptyBlock()

	p.MsgLog = make([]ptypes.MsgsLog, 64)

	// If there is no request to redo, start the normal process directly
	if len(msg.OSet) == 0 {
		p.PTimer.Timer.Stop()
		// p.View.NextLeaderName()
		p.CurPhase = ptypes.NEW_VIEW
		p.PTimer.Timer.Start(func() {
			// p.Logger.Println("[TIMER-EXPIRE-HandleNewview]:", p.GetNodeName(), "View:", p.View.ViewNumber)
			p.StartViewChange()
		}, func() {
			// fmt.Println("vcprepare timer stop")
		})
		return &ptypes.PMsg{
			MType: ptypes.VC_REPLY,
		}
	}

	// generate vc-prepare message
	vcPrepareMsg := &ptypes.PMsg{
		MType:      ptypes.VC_PREPARE,
		ViewNumber: p.View.ViewNumber,
		SendNode:   p.GetNodeName(),
		ReciNode:   "Broadcast",
		OSet:       make([]*ptypes.PMsg, 0),
	}

	// generate a prepare message for each valid preprepare message and it add it to the vc-prepare message after signing
	for _, m := range msg.OSet {
		prepareMsg := &ptypes.PMsg{
			MType:      ptypes.PREPARE,
			ViewNumber: p.View.ViewNumber,
			SeqNum:     m.SeqNum,
			SendNode:   p.GetNodeName(),
			Digest:     m.Digest,
			ReciNode:   "Broadcast",
		}

		sign := p.Signer.Sign(prepareMsg.Message2Byte(1))
		prepareMsg.Signature = sign
		vcPrepareMsg.OSet = append(vcPrepareMsg.OSet, prepareMsg)
	}

	// sign the vc-prepare message and add it
	sign := p.Signer.Sign(vcPrepareMsg.Message2Byte(1))
	vcPrepareMsg.Signature = sign

	// log
	p.Logger.Println("[VC-PREPARE]:", p.GetNodeName(), "View:", p.View.ViewNumber)

	return vcPrepareMsg
}

// Preprepare(): the node generate a new block
func (p *PBFT) Preprepare() *ptypes.PMsg {

	// p.ProposalLock.Lock()
	// defer p.ProposalLock.Unlock()

	// waiting new request
	if p.CurProposal.IsEmpty() {
		return nil
	}

	if p.CurPhase != ptypes.NEW_VIEW {
		return nil
	}

	// generate a new block with proposal's command
	p.BlkStore.GenNewBlock(p.View.ViewNumber, common.TwoDimByteSlice2StringSlice(p.CurProposal.Command))

	// generate pre-prepare message with new block, proposal, digest of block
	// sign for it and add it to message
	prePrepareMsg := &ptypes.PMsg{
		MType:      ptypes.PREPREPARE,
		ViewNumber: p.View.ViewNumber,
		SeqNum:     p.SequenceNum,
		Digest:     p.BlkStore.CurBlkHash,
		SendNode:   p.GetNodeName(),
		ReciNode:   "Broadcast",
		Proposal:   p.CurProposal,
		Block:      p.BlkStore.CurProposalBlk,
	}
	sign := p.Signer.Sign(prePrepareMsg.Message2Byte(0))
	prePrepareMsg.Signature = sign

	// log the pre-prepare message to local because the leader don't recieve self pre-prepare message
	p.LogMsg(prePrepareMsg)

	// update local phase
	p.CurPhase = ptypes.PREPREPARE

	// log
	// p.Logger.Println("[PRE-PREPARE]:", p.GetNodeName(), "View:", p.View.ViewNumber, len(prePrepareMsg.Proposal.Command), len(p.BlkStore.CurProposalBlk.BlkData.Trans), len(prePrepareMsg.Block.BlkData.Trans), p.CurProposal.Command[0][:2])

	return prePrepareMsg
}
