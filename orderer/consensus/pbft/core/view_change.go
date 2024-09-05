package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	ptypes "pbft/types"
)

// StartViewChange: start the view-change protocol when the timer expire
// StartViewChange implement PBFT description as follow:
// If the timer of backup i expires in view v, the backup starts a view change to move the system to
// view v+1. It stops accepting messages (other than checkpoint, view-change, and new-view messages) and
// multicasts a <VIEW-CHANGE, v+1, n, C, P, i>_i message to all replicas. Here n is the sequence number of the last
// stable checkpoint known to i, C is a set of 2f+1 valid checkpoint messages proving the correctness of s, and
// Pm is a set containing a set Pm for each request m that prepared at with a sequence number higher than n.
// Each set Pm contains a valid pre-prepare message (without the corresponding client message) and 2f matching, valid
// prepare messages signed by different backups with the same view, sequence number, and the digest of m.
func (p *PBFT) StartViewChange() {

	// when start view-change protocol, update local phase and refuse unconcerned messages
	// PTimer.VCMsgSendFlag indicates whether the node sends view-change messages
	p.CurPhase = ptypes.VIEW_CHANGE
	p.PTimer.VCMsgSendFlag = true

	// generate the new view-change message and sign it
	viewChangeMsg := ptypes.PMsg{
		MType:      ptypes.VIEW_CHANGE,
		ViewNumber: p.View.ViewNumber + 1,
		SeqNum:     p.CheckPoint.Seq,
		CSet:       p.CheckPoint.CPMsgs,
		PSet:       p.GetValidMsgs(),
		SendNode:   p.GetNodeName(),
		ReciNode:   "Broadcast",
	}
	sign := p.Signer.Sign(viewChangeMsg.Message2Byte(2))
	viewChangeMsg.Signature = sign

	// the view change message should be sent,and it's not recieved
	p.PTimer.VCMsgSendFlag = true

	// submit the view-change message to node and the node will broadcast it
	vCMsgJson, err := json.Marshal(viewChangeMsg)
	if err == nil {
		if p.ForwardChan != nil {
			p.ForwardChan <- vCMsgJson
		}
		if p.SendChan != nil {
			p.SendSerMsg(&viewChangeMsg)
		}
	}

	// log
	p.Logger.Println("[VIEW-CHANGE-START]:", p.GetNodeName(), "Expect enter view", p.View.ViewNumber+1)
}

// HandleViewChangeMsg: the node handle view-change messages and generate new-view message
// HandleViewChangeMsg implement PBFT description as follow:
// When the primary of view v+1 receives 2f+1 valid view-change messages for view v+1 from other replicas,
// it multicasts a <NEW-VIEW, v+1, V, O>_p message to all other replicas, where is a set containing the valid
// view-change messages received by the primary plus the view-change message for v+1 the primary sent (or would have sent), \
// and O is a set of pre-prepare messages (without the piggybacked request).
func (p *PBFT) HandleViewChangeMsg(msg *ptypes.PMsg) *ptypes.PMsg {

	// verify signature and log it
	if !p.Signer.VerifySign(msg.SendNode, msg.Signature, msg.Message2Byte(2)) {
		return nil
	}
	p.LogMsg(msg)

	// check the threshold, the equality here is to prevent multiple response messages
	if len(p.ViewChangeMsgs.NewViewMsgs) != (p.View.NodesNum-1)/3*2+1 {
		return nil
	}

	for p.View.ViewNumber < msg.ViewNumber {
		p.View.NextView()
	}

	// fmt.Println(p.GetNodeName(), p.View)
	// the replica only stop timer after recieve enough message
	if !p.IsLeader() {
		p.Logger.Println("[NEW-VIEW]:", p.GetNodeName(), "View:", p.View.ViewNumber)
		return nil
	}

	// generate VSet, which is the set of view-change message
	VSet := make([]*ptypes.PMsg, len(p.ViewChangeMsgs.NewViewMsgs))
	copy(VSet, p.ViewChangeMsgs.NewViewMsgs)

	// generate OSet, which is the set of preprepare with the sequence between min-s and max-s
	OSet := make([]*ptypes.PMsg, 0)

	// get the min-s(sequence of the last stable checkpoint) and max-s(the max sequence of valid pre-prepare message)
	minS, maxS := p.GetSeq()

	// generate new pre-prepare messages for each valid pre-prepare message after checkpoint
	// sign them respectively and add them to new-view message OSet
	if minS != -1 && maxS != -1 {
		for i := minS; i <= maxS; i++ {
			prePrepareMsg := &ptypes.PMsg{
				MType:      ptypes.PREPREPARE,
				ViewNumber: p.View.ViewNumber,
				SeqNum:     i,
			}

			// add the same sequence digest of PSet of some view-change message in VSet
		outloop:
			for _, vCMsg := range p.ViewChangeMsgs.NewViewMsgs {
				for _, pm := range vCMsg.PSet {
					if pm.PrePrepareMsg.SeqNum == i && len(pm.PrepareMsgs) > (p.View.NodesNum-1)/3*2 {
						prePrepareMsg.Digest = pm.PrePrepareMsg.Digest
						// fmt.Println(prePrepareMsg.Block)
						break outloop
					}
				}
			}

			sign := p.Signer.Sign(prePrepareMsg.Message2Byte(0))
			prePrepareMsg.Signature = sign
			OSet = append(OSet, prePrepareMsg)
		}
	}

	// generate new-view message and sign it
	newViewMsg := ptypes.PMsg{
		MType:      ptypes.NEW_VIEW,
		ViewNumber: p.View.ViewNumber,
		VSet:       VSet,
		OSet:       OSet,

		ReciNode: "Broadcast",
	}
	sign := p.Signer.Sign(newViewMsg.Message2Byte(3))
	newViewMsg.Signature = sign

	// log
	p.Logger.Println("[NEW-VIEW]:", p.GetNodeName(), "View:", p.View.ViewNumber, minS, maxS, len(OSet))

	return &newViewMsg
}

// GetSeq: get min-s and max-s from view-change messages' checkpoint sequence and prepare message sequence
// GetSeq implement PBFT description as follow:
// The primary determines the sequence number min-s of the latest stable checkpoint in V and
// the highest sequence number max-s in a prepare message in V.
func (p *PBFT) GetSeq() (int, int) {
	if len(p.ViewChangeMsgs.NewViewMsgs) == 0 || len(p.ViewChangeMsgs.NewViewMsgs[0].CSet) == 0 {
		fmt.Println("GetSeq eixst zero", p.GetNodeName(), len(p.ViewChangeMsgs.NewViewMsgs), len(p.ViewChangeMsgs.NewViewMsgs[0].CSet))
		return -1, -1
	}
	minS := -1
	maxS := -1

	for i := 0; i < len(p.ViewChangeMsgs.NewViewMsgs); i++ {
		for j := 0; j < len(p.ViewChangeMsgs.NewViewMsgs[i].CSet); j++ {
			if minS < p.ViewChangeMsgs.NewViewMsgs[i].CSet[j].SeqNum {
				minS = p.ViewChangeMsgs.NewViewMsgs[i].CSet[j].SeqNum
			}
		}
		for k := 0; k < len(p.ViewChangeMsgs.NewViewMsgs[i].PSet); k++ {
			for l := 0; l < len(p.ViewChangeMsgs.NewViewMsgs[i].PSet[k].PrepareMsgs); l++ {
				if maxS < p.ViewChangeMsgs.NewViewMsgs[i].PSet[k].PrepareMsgs[l].SeqNum {
					maxS = p.ViewChangeMsgs.NewViewMsgs[i].PSet[k].PrepareMsgs[l].SeqNum
				}
			}
		}
	}
	return minS, maxS
}

// VerifyOSet: the replica verify the OSet of the new-view message
func (p *PBFT) VerifyOSet(msg *ptypes.PMsg) bool {

	// the replica recreate O similar to the leader
	// myOSet := make([]*ptypes.PMsg, 0)
	minS, maxS := p.GetSeq()

	// when msg.OSet is empty reture true if and only if the all view-change messages' PSet is empty
	// otherwise reture false
	if len(msg.OSet) == 0 {
		for _, vCMsg := range p.ViewChangeMsgs.NewViewMsgs {
			if len(vCMsg.PSet) != 0 {
				fmt.Println("len(msg.OSet) == 0, len(vCMsg.PSet) != 0", vCMsg.SendNode)
				return false
			}
		}
		return true
	}

	if len(msg.OSet) != maxS-minS+1 {
		fmt.Println("len(msg.OSet) != maxS-minS+1", len(msg.OSet), minS, maxS, maxS-minS+1)
		return false
	}

	j := 0
	for i := minS; i < maxS; i++ {
		for _, vCMsg := range p.ViewChangeMsgs.NewViewMsgs {
			for _, pm := range vCMsg.PSet {
				if pm.PrePrepareMsg.SeqNum == i && len(pm.PrepareMsgs) > (p.View.NodesNum-1)/3*2 {
					if !bytes.Equal(msg.OSet[j].Digest, pm.PrePrepareMsg.Digest) || !p.Signer.VerifySign(msg.SendNode, msg.OSet[i].Signature, msg.OSet[i].Message2Byte(0)) {
						return false
					}
				}
			}
		}
	}
	return true
}
