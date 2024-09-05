package core

import (
	hstypes "hotstuff/types"
)

// HandlePreCommitVote: the leader in commit phase handle the message
// HandlePreCommitVote implement basic hotstuff description as follow:
// as a leader
// wait for (n − f) votes:
//
//	V ← {v | matchingMsg(v, pre-commit, curView)}
//
// precommitQC ← QC(V)
// broadcast Msg(commit, ⊥, precommitQC)
func (bhs *BCHotstuff) HandlePreCommitVote(msg *hstypes.Msg) *hstypes.Msg {

	// check the node whether in commit phase after successful pre-commit phase
	if bhs.CurPhase != hstypes.PRE_COMMIT {
		return nil
	}

	// check whether this node is leader
	if bhs.GetNodeName() != bhs.GetLeaderName() {
		return nil
	}

	// check whether message's type and view number are matching current view
	if bhs.MatchingMsg(msg, hstypes.PRE_COMMIT_VOTE, bhs.View.ViewNumber) {
		bhs.PreCommitVotes = append(bhs.PreCommitVotes, msg)
	}

	// check meet the threshold conditions, (m > 2f+1)
	if len(bhs.PreCommitVotes) <= (bhs.View.NodesNum-1)/3*2 {
		return nil
	}

	// generate lockQC
	// combine the part signature, and add it to the lockQC
	lockQC := hstypes.QC{
		QType:      hstypes.PRE_COMMIT,
		ViewNumber: msg.ViewNumber,
		HsNode:     msg.HsNode,
	}
	preCommitSig := bhs.CombineSign(&bhs.PreCommitVotes)
	if preCommitSig == nil {
		return nil
	}
	lockQC.Sign = preCommitSig

	// update the local lockQC
	// change the local phase to commit
	bhs.LockedQC = lockQC
	bhs.CurPhase = hstypes.COMMIT

	// log
	// bhs.Logger.Println("[COMMIT]", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber)

	// return the commit message with lockQC from leader
	// then the message will be broadcast to all replica
	return &hstypes.Msg{
		MType:      hstypes.COMMIT,
		ViewNumber: bhs.View.ViewNumber,
		Justify:    bhs.LockedQC,
		ReciNode:   "Broadcast",
	}
}

// HandleCommit: the replica in pre-commit phase handle the message
// HandleCommit implement basic hotstuff description as follow:
// as a replica
// wait for message m from leader(curView)
//
//	m : matchingQC(m.justify, pre-commit, curView)
//
// lockedQC ← m.justify
// send to leader(curView)
// voteMsg(commit, m.justify.node, ⊥)
func (bhs *BCHotstuff) HandleCommit(msg *hstypes.Msg) *hstypes.Msg {
	// check whether the node state is commit phase after pre-commit phase
	if bhs.CurPhase != hstypes.PRE_COMMIT {
		return nil
	}

	// if the node is leader, ignore this commit message
	if bhs.GetNodeName() == bhs.GetLeaderName() {
		return nil
	}

	// check whether message is from this view
	if msg.ViewNumber != bhs.View.ViewNumber {
		return nil
	}

	// check whether message's QC match pre-commit
	if !bhs.CheckQC(msg, hstypes.PRE_COMMIT, bhs.View.ViewNumber) {
		return nil
	}

	// update local lockQC
	bhs.LockedQC = msg.Justify

	// generate the commit vote and sign for it
	comVote := hstypes.Msg{
		MType:      hstypes.COMMIT,
		ViewNumber: msg.ViewNumber,
		HsNode:     msg.Justify.HsNode,
		ReciNode:   bhs.GetLeaderName(),
	}
	sigMsg := comVote.Message2Byte()
	partSig, err := bhs.ThresholdSigner.ThresholdSign(sigMsg)
	if err != nil {
		return nil
	}

	// update the vote type and part signature
	comVote.PartialSig = partSig
	comVote.MType = hstypes.COMMIT_VOTE

	// change the node local state to commit and log
	bhs.CurPhase = hstypes.COMMIT
	// bhs.Logger.Println("[COMMIT]", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber, bhs.CurProposal.Commands[0][:10])

	// return the vote and unicast to leader
	return &comVote
}
