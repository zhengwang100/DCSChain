package core

import (
	hs2types "hotstuff2/types"
)

// HandleVote1: the leader collect 2f+1 vote1 and combine them signature
// HandleVote1 implement hotstuff-2 description as follow:
// Prepare. Upon receiving 2𝑡 + 1 votes for block 𝐵𝑘 , the leader forms certificate 𝐶𝑣 (𝐵𝑘 )
// and broadcasts a request ⟨prepare, 𝐶𝑣 (𝐵𝑘 )⟩𝐿𝑣 to all parties.
func (hs2 *Hotstuff2) HandleVote1(msg *hs2types.H2Msg) *hs2types.H2Msg {

	// check whether message view number is matching the node view number
	if msg.ViewNumber != hs2.View.ViewNumber {
		return nil
	}
	hs2.Vote1 = append(hs2.Vote1, msg)

	// check the local phase
	if hs2.CurPhase != hs2types.PROPOSE {
		return nil
	}

	// check threshold
	if len(hs2.Vote1) <= (hs2.View.NodesNum-1)/3*2 {
		return nil
	}

	// generate the proposal QC
	proposalQC := hs2types.QuromCert{
		QType:      hs2types.PROPOSE,
		ViewNumber: hs2.View.ViewNumber,
		Hs2Node:    hs2.CurHs2Node,
		Height:     hs2.BlkStore.CurProposalBlk.BlkHdr.Height,
	}

	// recover the signed message
	msgSign := hs2types.H2Msg{
		MType:      hs2types.PROPOSE,
		ViewNumber: hs2.View.ViewNumber,
		Hs2Node:    hs2.CurHs2Node,
	}

	// combine the part signature, and add it to the proposalQC
	proposalSign := hs2.CombineSign(hs2.Vote1, msgSign)
	proposalQC.Sign = proposalSign

	// update the local proposalQC
	// change the local phase to prepare
	hs2.ProposalQC = proposalQC
	hs2.CurPhase = hs2types.PREPARE

	// log
	// hs2.Logger.Println("[PREPARE]:", hs2.GetNodeName(), "Succeed!")

	return &hs2types.H2Msg{
		MType:      hs2types.PREPARE,
		ViewNumber: hs2.View.ViewNumber,
		Hs2Node:    hs2.CurHs2Node,
		ReciNode:   "Broadcast",
		Justify1:   proposalQC,
	}
}
