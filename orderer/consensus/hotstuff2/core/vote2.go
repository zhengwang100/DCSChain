package core

import (
	hs2types "hotstuff2/types"
)

// HandlePrepare: the replace the prepare message and sign for it
// HandlePrepare implement hotstuff-2 description as follow:
// Vote2. Upon receiving ⟨prepare, 𝐶𝑣 (𝐵𝑘 )⟩𝐿𝑣 , a party updates their lock to 𝐵𝑘 and the locked certificate to 𝐶𝑣 (𝐵𝑘 ).
// It sends ⟨vote2, 𝐶𝑣 (𝐵𝑘 ), 𝑣⟩ to 𝐿𝑣+1.
func (hs2 *Hotstuff2) HandlePrepare(msg *hs2types.H2Msg) *hs2types.H2Msg {
	// check whether message view number is matching the node view number
	if hs2.View.ViewNumber < msg.ViewNumber {
		return nil
	}

	// when the leader is not in prepare phase and the replica is not in propose phase, return nil
	if (hs2.IsLeader() && hs2.CurPhase != hs2types.PREPARE) || (!hs2.IsLeader() && hs2.CurPhase != hs2types.PROPOSE) {
		return nil
	}

	// check 𝐶𝑣(𝐵𝑘),single authentication block Bk type is propose
	if !hs2.CheckQC(&msg.Justify1, hs2types.PROPOSE, hs2.View.ViewNumber, hs2.BlkStore.CurProposalBlk.BlkHdr.Height) {
		return nil
	}

	// generate the vote1 for the new proposal, and sign for it
	vote2 := &hs2types.H2Msg{
		MType:      msg.MType,
		ViewNumber: msg.ViewNumber,
		Hs2Node:    msg.Hs2Node,
	}
	partSign, err := hs2.ThresholdSigner.ThresholdSign(vote2.Message2Byte())
	if err != nil {
		return nil
	}
	vote2.ConsSign = partSign
	vote2.ReciNode = hs2.GetNextLeaderName()
	vote2.MType = hs2types.VOTE2

	// in general, until now the round has finished for itself
	// and it will stop the ViewTimer
	hs2.PM.ViewTimer.Stop()

	// update local proposal QC
	hs2.ProposalQC = msg.Justify1
	hs2.CurPhase = hs2types.PREPARE

	// log
	// hs2.Logger.Println("[VOTE2]:", hs2.GetNodeName(), hs2.View.ViewNumber, " Succeed!")

	return vote2
}

// HandleVote2: the next leader handle the vote2 from last view
func (hs2 *Hotstuff2) HandleVote2(msg *hs2types.H2Msg) *hs2types.H2Msg {

	// check whether message view number is matching the node view number
	if msg.ViewNumber < hs2.View.ViewNumber {
		return nil
	}

	hs2.Vote2 = append(hs2.Vote2, msg)
	if hs2.CurPhase != hs2types.PREPARE {
		return nil
	}

	// check threshold
	if len(hs2.Vote2) <= (hs2.View.NodesNum-1)/3*2 {
		return nil
	}

	// generate the prepare QC
	prepareQC := hs2types.QuromCert{
		QType:      hs2types.PREPARE,
		ViewNumber: hs2.View.ViewNumber,
		Hs2Node:    hs2.CurHs2Node,
		Height:     hs2.BlkStore.CurProposalBlk.BlkHdr.Height,
	}

	// recover the signed message
	msgSign := hs2types.H2Msg{
		MType:      hs2types.PREPARE,
		ViewNumber: hs2.View.ViewNumber,
		Hs2Node:    hs2.CurHs2Node,
	}

	// combine the part signature, and add it to the proposalQC
	prepareSign := hs2.CombineSign(hs2.Vote2, msgSign)
	if prepareSign == nil {
		return nil
	}
	prepareQC.Sign = prepareSign

	// update the local proposalQC
	// change the local phase to NEW-VIEW
	hs2.PrepareQC = prepareQC
	hs2.CurPhase = hs2types.NEW_VIEW

	// log
	// hs2.Logger.Println("[COLLECT-VOTE2]:", hs2.GetNodeName(), "Succeed!")
	// hs2.NewRound()

	return &hs2types.H2Msg{
		MType:      hs2types.ENTER,
		ViewNumber: hs2.View.ViewNumber,
		ReciNode:   "Broadcast",
		Justify2:   prepareQC,
	}
}

// NewRound: start a new round consensus, refresh the consensus state
func (hs2 *Hotstuff2) NewRound() {
	// hs2.CurPhase = hstypes.NEW_VIEW
	hs2.View.NextView()
	hs2.LastRoundMsg = hs2.CurRoundMsgs
	hs2.CurRoundMsgs = make([]*hs2types.H2Msg, 0)
	hs2.NewViewMsgs = make([]*hs2types.H2Msg, 0)
	hs2.Vote1 = make([]*hs2types.H2Msg, 0)
	hs2.Vote2 = make([]*hs2types.H2Msg, 0)
}
