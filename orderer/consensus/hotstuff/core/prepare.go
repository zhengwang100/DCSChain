package core

import (
	hstypes "hotstuff/types"
)

// HandlePrepare: the replica in prepare phase handle the message
// HandlePrepare implement basic hotstuff description as follow:
// wait for message m from leader(curView)
//
//	m : matchingMsg(m, prepare, curView)
//
// if m.node extends fromm.justify.node ∧  safeNode(m.node, m.justify) then
// send voteMsg(prepare, m.node, ⊥) to leader(curView)
func (bhs *BCHotstuff) HandlePrepare(msg *hstypes.Msg, pk []byte) *hstypes.Msg {

	// check whether the node state is new-view phase
	if bhs.CurPhase != hstypes.NEW_VIEW {
		// bhs.Logger.Println("[ERROR]:", bhs.GetNodeName(), "current phase is not new-view")
		return nil
	}

	// if the node is leader, ignore this prepare message
	if bhs.GetNodeName() == bhs.GetLeaderName() {
		return nil
	}

	if bhs.CurProposal.IsEmpty() {
		bhs.CurProposal = msg.Proposal
	}

	// check whether message is from this view
	if msg.ViewNumber != bhs.View.ViewNumber {
		return nil
	}

	// check whether this message's node is valid
	if bhs.View.ViewNumber != 0 && !bhs.CheckNewHsNode(msg) {
		return nil
	}

	if !bhs.VerifyReqs(msg.Proposal.Commands, msg.Proposal.Signs, pk) {
		return nil
	}

	bhs.ViewTimer.Stop()

	// generate the vote for the message, sign for the recieved prepare message and get this node's part signature
	// this new message is easy to unify the undersigned message
	// message -> byte -> signature
	prepareVote := hstypes.Msg{
		MType:      msg.MType,
		ViewNumber: msg.ViewNumber,
		HsNode:     msg.HsNode,
		ReciNode:   bhs.GetLeaderName(),
	}
	sigMsg := prepareVote.Message2Byte()
	partSig, err := bhs.ThresholdSigner.ThresholdSign(sigMsg)
	if err != nil {
		return nil
	}

	// update the vote type and part signature
	prepareVote.MType = hstypes.PREPARE_VOTE
	prepareVote.PartialSig = partSig

	// change the node local state to prepare
	// accept(store) the new proposal, update local current proposal
	bhs.CurPhase = hstypes.PREPARE
	bhs.BlkStore.CurProposalBlk = msg.Block
	bhs.BlkStore.CurBlkHash = bhs.BlkStore.CurProposalBlk.Hash()

	// log
	// bhs.Logger.Println("[PREPARE]", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber)

	bhs.ViewTimer.Start(func() {
		// bhs.Logger.Println("prepare View Timer expire", bhs.GetNodeName())
		bhs.StartViewChange()
	}, func() {
		// bhs.Logger.Println("prepare View Timer stop", bhs.GetNodeName())
	})

	// return the vote and unicast to leader

	return &prepareVote
}
