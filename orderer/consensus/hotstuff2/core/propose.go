package core

import (
	common "common"
	hs2types "hotstuff2/types"
)

// GenProposal: the leader propose a new proposal
// GenProposal implement hotstuff-2 description as follow:
// The leader 𝐿_𝑣 broadcasts ⟨ propose, 𝐵_𝑘, 𝑣, 𝐶_𝑣'(𝐵_𝑘−1), 𝐶_𝑣"(𝐶_𝑣"(𝐵_𝑘"))⟩_L_v
// Here, 𝐵𝑘 := (𝑏𝑘, ℎ𝑘−1) is the block that should extend the highest certified block 𝐵𝑘−1
// with certificate 𝐶𝑣′ (𝐵𝑘−1) known to leader and 𝐶𝑣′′ (𝐶𝑣′′ (𝐵𝑘′′ )) is
// the highest double certificate known to the leader.
// note: this func is executed by the leader
func (hs2 *Hotstuff2) GenProposal(msg *hs2types.H2Msg) *hs2types.H2Msg {
	hs2.ProposalLock.Lock()
	defer hs2.ProposalLock.Unlock()

	// check whether this node is right phase
	if hs2.CurPhase != hs2types.NEW_PROPOSE {
		return nil
	}

	// if proceeding directly to the propose step using 𝐶𝑣−1(𝐶𝑣−1(𝐵𝑘−1)), check the prepare QC
	// if it is valid, update local locked prepare QC and pacemaker OptimisticFlag which means process is normal
	if hs2.PM.OptimisticFlag && msg.Justify2.QType != hs2types.NEW_VIEW && hs2.CheckQC(&msg.Justify2, hs2types.PREPARE, msg.ViewNumber, msg.Justify2.Height) {
		hs2.PrepareQC = msg.Justify2
		hs2.PM.OptimisticFlag = false
	}

	// for simplicity, the consensus at the time of the experiment must propose the valid consensus,
	// and in practice, the system should receive relevant requests all the time.
	if hs2.CurProposal.IsEmpty() {
		return nil
	}

	// for the new block height the leader add validation and store block before generate a new block
	hs2.AddValidation2Blk(hs2.PrepareQC)
	hs2.WriteNewH2Block(hs2.GenCommitBlkIndex())

	// generate a new block, update local proposal in this view
	hs2.BlkStore.GenNewBlock(hs2.View.ViewNumber, common.TwoDimByteSlice2StringSlice(hs2.CurProposal.Command))
	hs2.CurProposal.ViewNumber = hs2.View.ViewNumber
	// extend from the longest chain or last block which is the known height block in system
	hs2.CurHs2Node = hs2.CreateLeaf(hs2.ProposalQC.Hs2Node.CurHash)

	// generate propose message with new block and the known height certification
	proposalMsg := &hs2types.H2Msg{
		MType:      hs2types.PROPOSE,
		ViewNumber: hs2.View.ViewNumber,
		SendNode:   hs2.GetNodeName(),
		ReciNode:   "Broadcast",
		Block:      hs2.BlkStore.CurProposalBlk,
		Hs2Node:    hs2.CurHs2Node,
		Justify1:   hs2.ProposalQC,
		Justify2:   hs2.PrepareQC,
	}

	// update local phase
	hs2.CurPhase = hs2types.PROPOSE

	// log
	// hs2.Logger.Println("[PROPOSE]:", hs2.GetNodeName(), "in view", hs2.View.ViewNumber, "Succeed!")

	return proposalMsg
}

// GetHighQCIndex: from some messages, return the index of message with the highest certified block
// params: recieved new-view messages
// return: the index of the highest one certified block and double certificate
func GetHighestCertIndex(newViewMsgs []*hs2types.H2Msg) (int, int) {
	msgNum := len(newViewMsgs)
	maxIndex1 := 0
	maxIndex2 := 0
	maxViewNumber1 := (newViewMsgs)[0].Justify1.ViewNumber
	maxViewNumber2 := (newViewMsgs)[0].Justify2.ViewNumber
	for i := 0; i < msgNum; i++ {
		if (newViewMsgs)[i].Justify1.ViewNumber > maxViewNumber1 {
			maxIndex1 = i
			maxViewNumber1 = (newViewMsgs)[i].Justify1.ViewNumber
		}
		if (newViewMsgs)[i].Justify2.ViewNumber > maxViewNumber2 {

			maxIndex2 = i
			maxViewNumber2 = (newViewMsgs)[i].Justify1.ViewNumber
		}
	}
	return maxIndex1, maxIndex2
}
