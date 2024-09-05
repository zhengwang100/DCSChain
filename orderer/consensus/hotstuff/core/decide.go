package core

import (
	hstypes "hotstuff/types"
)

// HandleCommitVote: the leader in decide phase handle the message
// HandleCommitVote implement basic hotstuff description as follow:
// as a leader
// wait for (n − f) votes:
//
//	V ← {v | matchingMsg(v, commit, curView)}
//
// commitQC ← QC(V)
// broadcast Msg(decide, ⊥, commitQC)
func (bhs *BCHotstuff) HandleCommitVote(msg *hstypes.Msg) *hstypes.Msg {
	// check the node whether in decide phase after successful commit phase
	if bhs.CurPhase != hstypes.COMMIT {
		return nil
	}

	// check whether this node is leader
	if bhs.GetNodeName() != bhs.GetLeaderName() {
		return nil
	}

	// check whether message's type and view number are matching current view
	if bhs.MatchingMsg(msg, hstypes.COMMIT_VOTE, bhs.View.ViewNumber) {
		bhs.CommitVotes = append(bhs.CommitVotes, msg)
	}

	// check meet the threshold conditions, (m > 2f+1)
	if len(bhs.CommitVotes) <= (bhs.View.NodesNum-1)/3*2 {
		return nil
	}

	// generate commitQC
	// combine the part signature, and add it to the commitQC
	commitQC := hstypes.QC{
		QType:      hstypes.COMMIT,
		ViewNumber: msg.ViewNumber,
		HsNode:     msg.HsNode,
	}
	commitSig := bhs.CombineSign(&bhs.CommitVotes)
	if commitSig == nil {
		return nil
	}
	commitQC.Sign = commitSig

	// different from the previous stage, this phase don't need to change local state
	// and the leader also execute HandleDecide in which will change and refresh local state

	// return the decide message with prepareQC from leader
	// then the message will be broadcast to all replica
	return &hstypes.Msg{
		MType:      hstypes.DECIDE,
		ViewNumber: bhs.View.ViewNumber,
		Justify:    commitQC,
		ReciNode:   "Broadcast",
	}
}

// HandleDecide: the all replica include the leader (only) in decide phase handle the message
// HandleDecide implement basic hotstuff description as follow:
// as a replica
// wait for message m from leader(curView)
//
//	m : matchingQC(m.justify, commit, curView)
//
// execute new commands through m.justify.node,
// ..respond to clients
func (bhs *BCHotstuff) HandleDecide(msg *hstypes.Msg) *hstypes.Msg {
	// fmt.Println(bhs.GetNodeName(), bhs.View.ViewNumber)

	// check the node whether in decide phase after successful commit phase
	if bhs.CurPhase != hstypes.COMMIT {
		return nil
	}

	// check whether message is from this view
	if msg.ViewNumber != bhs.View.ViewNumber {
		return nil
	}

	// check whether message's QC match commit
	if !bhs.CheckQC(msg, hstypes.COMMIT, bhs.View.ViewNumber) {
		return nil
	}

	// stop the view timer because the view has succeed!
	bhs.ViewTimer.Stop()

	// change the node local state to decide and log
	bhs.CurPhase = hstypes.DECIDE
	// bhs.Logger.Println("[DECIDE]", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber, len(bhs.BlkStore.CurProposalBlk.BlkData.Trans))

	// add validation to the block and store it
	bhs.BlkStore.CurProposalBlk.BlkHdr.Validation = msg.Justify.Sign
	bhs.BlkStore.StoreBlock(bhs.BlkStore.CurProposalBlk)

	// refresh the local consensus state include view update
	bhs.NewRound()

	// generate new-view message
	// at this point, the consensus process is basically complete except for executing the command
	newMsg := &hstypes.Msg{
		MType:      hstypes.NEW_VIEW,
		ViewNumber: bhs.View.ViewNumber,
		SendNode:   bhs.GetNodeName(),
		Justify:    bhs.PrepareQC,
		ReciNode:   bhs.GetLeaderName(),
	}

	bhs.ViewTimer.Start(func() {
		// bhs.Logger.Println("decide View Timer expire", bhs.GetNodeName())
		bhs.StartViewChange()
	}, func() {
		// bhs.Logger.Println("decide View Timer stop", bhs.GetNodeName())
	})

	return newMsg
}

// NewRound: start a new round consensus, refresh the consensus state
func (bhs *BCHotstuff) NewRound() {
	// bhs.CurPhase = hstypes.NEW_VIEW
	bhs.View.NextView()
	bhs.LastRoundMsg = bhs.CurRoundMsg
	bhs.CurRoundMsg = make([]*hstypes.Msg, 0)
	bhs.NewViewMsgs = make([]*hstypes.Msg, 0)
	bhs.PrepareVotes = make([]*hstypes.Msg, 0)
	bhs.PreCommitVotes = make([]*hstypes.Msg, 0)
	bhs.CommitVotes = make([]*hstypes.Msg, 0)
}
