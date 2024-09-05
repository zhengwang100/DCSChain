package core

import (
	"bytes"
	"fmt"
	hs2types "hotstuff2/types"
)

// HandlePropose: the leader handle propose message which in fact is to generate a new proposal after TC or double certificate
// HandlePropose implement hotstuff-2 description as follow:
// Vote and commit. Upon receiving the first valid proposal ⟨propose, 𝐵𝑘, 𝑣, 𝐶𝑣′ (𝐵𝑘−1)⟩𝐿𝑣 in view 𝑣:
// • If 𝐶𝑣′ (𝐵𝑘−1) is ranked no lower than the locked block, then send ⟨vote, 𝐵𝑘, 𝑣⟩ as a threshold signature share to 𝐿𝑣.
// Update lock to 𝐵𝑘−1 and the certificate to 𝐶𝑣′ (𝐵𝑘−1).
// • The party commits block 𝐵𝑘′′ and all its ancestors.
func (hs2 *Hotstuff2) HandlePropose(msg *hs2types.H2Msg) *hs2types.H2Msg {
	if hs2.CurPhase == hs2types.PROPOSE && !hs2.IsLeader() {
		return nil
	}
	// check view number
	if hs2.View.ViewNumber != msg.ViewNumber {
		return nil
	}

	// check 𝐶𝑣′ (𝐵𝑘−1) , single authentication block type is propose
	// the leader without the need to check itself message
	if hs2.IsLeader() && hs2.GetNodeName() == msg.SendNode {
		if hs2.View.ViewNumber != 0 && !hs2.IgnoreCheckQC &&
			!hs2.CheckQC(&msg.Justify1, hs2types.PROPOSE, msg.Justify1.ViewNumber, hs2.BlkStore.Height-1) {
			fmt.Println("Leader error", hs2.View.ViewNumber != 0, !hs2.CheckQC(&msg.Justify1, hs2types.PROPOSE, msg.Justify1.ViewNumber, hs2.BlkStore.Height-1))
			return nil
		}
	} else if hs2.View.ViewNumber != 0 && !hs2.IgnoreCheckQC &&
		!hs2.CheckQC(&msg.Justify1, hs2types.PROPOSE, msg.Justify1.ViewNumber, hs2.BlkStore.Height) {
		fmt.Println("Prepare 1", hs2.GetNodeName())
		return nil

	} else if hs2.ProposalQC.ViewNumber < msg.Justify1.ViewNumber {
		hs2.ProposalQC = msg.Justify1
	}

	// commit start
	// the replica check 𝐶𝑣"(𝐶𝑣"(𝐵𝑘")) namely double single authentication block
	// if valid add validation and store block
	if !hs2.IsLeader() {
		if hs2.View.ViewNumber != 0 && !hs2.IgnoreCheckQC &&
			!hs2.CheckQC(&msg.Justify2, hs2types.PREPARE, msg.Justify2.ViewNumber, msg.Justify2.Height) {
			fmt.Println("Prepare 2")
			return nil
		} else {
			hs2.AddValidation2Blk(msg.Justify2)
		}
		hs2.WriteNewH2Block(hs2.GenCommitBlkIndex())
	}
	// commit end

	// update local block, Hs2Node and view
	hs2.IgnoreCheckQC = false
	hs2.CurHs2Node = msg.Hs2Node
	hs2.BlkStore.CurProposalBlk = msg.Block
	hs2.BlkStore.CurBlkHash = hs2.BlkStore.CurProposalBlk.Hash()
	if msg.ViewNumber > hs2.View.ViewNumber {
		hs2.View.ViewNumber = msg.ViewNumber
	}

	// generate the vote1 for the new proposal, and sign for it
	vote1 := hs2types.H2Msg{
		MType:      msg.MType,
		ViewNumber: msg.ViewNumber,
		Hs2Node:    msg.Hs2Node,
		ReciNode:   msg.SendNode,
	}
	partSign, err := hs2.ThresholdSigner.ThresholdSign(vote1.Message2Byte())
	if err != nil {
		return nil
	}
	vote1.ConsSign = partSign
	vote1.ReciNode = msg.SendNode
	vote1.MType = hs2types.VOTE1

	// update local phase and log
	hs2.CurPhase = hs2types.PROPOSE
	// hs2.Logger.Println("[VOTE1]:", hs2.GetNodeName(), "Succeed!")

	return &vote1
}

// GenCommitBlkIndex: generate commit block index from local locked block and Hs2Node
// return: the waiting store block index [start, end]
func (hs2 *Hotstuff2) GenCommitBlkIndex() (int, int) {
	start, end := -1, -1
	count := len(hs2.LockBlk)
	for i := 0; i < count; i++ {
		if hs2.LockBlk[i].BlkHdr.ViewNumber == hs2.PrepareQC.ViewNumber {
			end = i
			break
		}
	}
	if end != -1 {
		start = end
		for i := end - 1; i >= 0; i-- {
			if !bytes.Equal(hs2.LockHs2Node[i].CurHash, hs2.LockHs2Node[i+1].ParentHash) {
				break
			}
			start--
		}
	}
	return start, end
}

// AddValidation2Blk: add validation to corresponding block for further verrify
func (hs2 *Hotstuff2) AddValidation2Blk(qc hs2types.QuromCert) {
	for _, blk := range hs2.LockBlk {
		if blk.BlkHdr.ViewNumber == qc.ViewNumber {
			blk.BlkHdr.Validation = qc.Sign
			break
		}
	}
}
