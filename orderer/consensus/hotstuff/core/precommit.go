package core

import (
	"fmt"
	hstypes "hotstuff/types"
)

// HandlePrepareVote: the leader in pre-commit phase handle the message
// HandlePrepareVote implement basic hotstuff description as follow:
// as a leader
// wait for (n − f) votes:
//
//	V ← {v | matchingMsg(v, prepare, curView)}
//
// prepareQC ← QC(V)
// broadcast Msg(pre-commit, ⊥, prepareQC)
func (bhs *BCHotstuff) HandlePrepareVote(msg *hstypes.Msg) *hstypes.Msg {
	// if bhs.View.NodesNum != 5 {
	// 	return nil
	// }

	// check the node whether in pre-commit phase after successful prepare phase
	if bhs.CurPhase != hstypes.PREPARE {
		// this node phase is not prepare
		return nil
	}

	// check whether this node is leader
	if bhs.GetNodeName() != bhs.GetLeaderName() {
		return nil
	}

	// check whether message's type and view number are matching current view
	if bhs.MatchingMsg(msg, hstypes.PREPARE_VOTE, bhs.View.ViewNumber) {
		bhs.PrepareVotes = append(bhs.PrepareVotes, msg)
	}

	// check meet the threshold conditions, (m > 2f+1)
	if len(bhs.PrepareVotes) <= (bhs.View.NodesNum-1)/3*2 {
		return nil
	}

	// generate prepareQC
	// combine the part signature, and add it to the prepareQC
	prepareQC := hstypes.QC{
		QType:      hstypes.PREPARE,
		ViewNumber: msg.ViewNumber,
		HsNode:     msg.HsNode,
	}
	prepareSig := bhs.CombineSign(&bhs.PrepareVotes)
	if prepareSig == nil {
		fmt.Println("Handle prepare-vote combine sign error", bhs.View.ViewNumber, bhs.GetNodeName())
		return nil
	}
	prepareQC.Sign = prepareSig

	// update the local prepareQC
	// change the local phase to pre-commit
	bhs.PrepareQC = prepareQC
	bhs.CurPhase = hstypes.PRE_COMMIT

	// log
	// bhs.Logger.Println("[PRE-COMMIT]", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber)

	// return the pre-commit message with prepareQC from leader
	// then the message will be broadcast to all replica
	return &hstypes.Msg{
		MType:      hstypes.PRE_COMMIT,
		ViewNumber: bhs.View.ViewNumber,
		Justify:    bhs.PrepareQC,
		ReciNode:   "Broadcast",
	}
}

// HandlePreCommit: the replica in pre-commit phase handle the message
// HandlePreCommit implement basic hotstuff description as follow:
// as a replica
// wait for message m from leader(curView)
//
//	m : matchingQC(m.justify, prepare, curView)
//
// prepareQC ← m.justify
// send to leader(curView)
// voteMsg(pre-commit, m.justify.node, ⊥)
func (bhs *BCHotstuff) HandlePreCommit(msg *hstypes.Msg) *hstypes.Msg {

	// fmt.Println(bhs.ConsId, bhs.GetLeaderName(), msg.SendNode)
	// check whether the node state is pre-commit phase after prepare phase
	if bhs.CurPhase != hstypes.PREPARE && bhs.CurPhase != hstypes.NEW_VIEW {
		return nil
	}

	// if the node is leader, ignore this precommit message
	if bhs.GetNodeName() == bhs.GetLeaderName() {
		return nil
	}

	// check whether message is from this view
	if msg.ViewNumber != bhs.View.ViewNumber {
		return nil
	}

	// check whether message's QC match prepare
	if !bhs.CheckQC(msg, hstypes.PREPARE, bhs.View.ViewNumber) {
		return nil
	}

	// update local hotstuff node and prepareQC
	bhs.HsNode = msg.Justify.HsNode
	bhs.PrepareQC = msg.Justify

	// generate the pre-commit vote and sign for it
	preComVote := hstypes.Msg{
		MType:      msg.MType,
		ViewNumber: msg.Justify.ViewNumber,
		HsNode:     msg.Justify.HsNode,
		ReciNode:   bhs.GetLeaderName(),
	}
	sigMsg := preComVote.Message2Byte()
	partSig, err := bhs.ThresholdSigner.ThresholdSign(sigMsg)
	if err != nil {
		return nil
	}

	// update the vote type and part signature
	preComVote.MType = hstypes.PRE_COMMIT_VOTE
	preComVote.PartialSig = partSig

	// change the node local state to pre-commit and log
	bhs.CurPhase = hstypes.PRE_COMMIT
	// bhs.Logger.Println("[PRE-COMMIT]", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber)

	// return the vote and unicast to leader
	return &preComVote
}
