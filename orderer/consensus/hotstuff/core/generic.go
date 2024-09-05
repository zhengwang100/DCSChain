package core

import (
	"bytes"
	common "common"
	"encoding/json"
	hstypes "hotstuff/types"
	"message"
	"strconv"
)

// CHandleNewView: the leader in generic phase handle the message
// CHandleNewView implement chained hotstuff description as follow:
// as a leader
// wait for (n − f) new-view messages:
//
//	M ← {m | matchingMsg(m, new-view, curView−1)}
//
// genericQC ← (arg max {m.justify.viewNumber}).justify
// curProposal ← createLeaf(genericQC.node, client's command, genericQC) ( m ∈ M )
// broadcast Msg(generic, curProposal, ⊥) # prepare phase (leader-half)
func (chs *CHotstuff) CHandleNewView(msg *hstypes.CMsg) []*hstypes.CMsg {

	// check whether the node state is new-view phase
	if chs.CurPhase != hstypes.NEW_VIEW {
		return nil
	}

	// check whether message's type and view number are matching current view
	if MatchingCMsg(msg, hstypes.NEW_VIEW, chs.View.ViewNumber) {

		// log new-view message
		chs.NewViewMsgs = append(chs.NewViewMsgs, msg)

		// estimate whether recieve the new-view message from the last leader
		if msg.SendNode == chs.GetChainedLastLeader() {
			chs.LastLeaderState = true
		}
	}

	// if the node isn't leader in the local view, ignore this message
	if !chs.IsLeader() {
		return nil
	}

	// check meet the threshold conditions, (m > 2f+1)
	if len(chs.NewViewMsgs) <= (chs.View.NodesNum-1)/3*2 {
		return nil
	}

	// if this leader hasn't recieved the new-view message from last leader, it will wait for it
	// then there's the timer mechanism, but now isn't.
	if !chs.LastLeaderState && !chs.ViewChangeFlag {
		return nil
	}

	// recover the two flags
	chs.LastLeaderState = false
	chs.ViewChangeFlag = false

	// log
	// chs.Logger.Println("[ENTER]", chs.GetNodeName(), "ViewNumber: "+strconv.Itoa(chs.View.ViewNumber))

	// let node submit recieved requests or create an empty proposal for the liveness
	// if ViewChangeFlag is ture, that triggers the view-change protocol after timer expired

	chs.CurPhase = hstypes.WAITING

	if chs.ViewChangeFlag {
		chs.CurPhase = hstypes.NEW_VIEW
		return nil
	} else if !chs.ExistNotExecuteBlock() {
		if chs.CurProposal.IsEmpty() {
			return nil
		}
	}

	if !chs.ProposalLock.TryLock() {
		return nil
	}
	defer chs.ProposalLock.Unlock()

	// stop view timer set in the previous "CHandleGeneric" func in last view
	chs.ViewTimer.Stop()

	// get the message with the hignest QC, and update local genericQC
	hignQcMsg, _ := GetHighChainedQC(&chs.NewViewMsgs)
	chs.GenericQC = hignQcMsg.Justify

	// create a new hotstuff node extend the hignest QC's node and local HsNode
	if len(chs.CurProposal.Commands) == 0 {
		chs.BlkStore.GenEmptyBlock()
		chs.BlkStore.Height += 1
	} else {
		reqs := common.CutOffTwoDimByteSlice(chs.CurProposal.Commands, 128)
		chs.BlkStore.GenNewBlock(chs.View.ViewNumber, common.TwoDimByteSlice2StringSlice(reqs), chs.BlkStore.GeneratedHeight)
		chs.BlkStore.Height += 1
	}

	// leader create leaf node extend from the hignest QC's node but doesn't update it to local CHsNode
	// leader update block to lock block
	newCHsNode := chs.CreateLeaf((hignQcMsg.Justify.HsNodes[0]).CurHash)
	msgCHsNode := [4]common.HsNode{newCHsNode, chs.HsNodes[0], chs.HsNodes[1], chs.HsNodes[2]}
	chs.UpdateBlock(&chs.BlkStore.CurProposalBlk)

	// update local state
	chs.CurPhase = hstypes.HALF_GENERIC
	chs.LastLeaderState = false
	chs.ExecuteState = false
	chs.NewViewMsgs = make([]*hstypes.CMsg, 0)
	chs.GenericVoteMsgs = make([]*hstypes.CMsg, 0)

	// log
	// chs.Logger.Println("[NEW_VIEW]", "r_"+strconv.Itoa(chs.ConsId)+" ViewNumber:", chs.View.ViewNumber, " Success!")

	return []*hstypes.CMsg{{
		MType:      hstypes.GENERIC,
		ViewNumber: chs.View.ViewNumber,
		ReciNode:   "Broadcast",
		HsNodes:    msgCHsNode,
		Proposal:   chs.CurProposal,
		Blk:        chs.BlkStore.CurProposalBlk,
		Justify:    chs.GenericQC,
	},
	}
}

// GenProposal: leader generates a new proposal and processes generic phase
// return:
// - the generic message
func (chs *CHotstuff) GenProposal() *hstypes.CMsg {

	// check meet the threshold conditions, (m > 2f+1)
	if len(chs.NewViewMsgs) <= (chs.View.NodesNum-1)/3*2 && chs.View.ViewNumber != 0 {
		return nil
	}

	// stop view timer set in the previous "CHandleGeneric" func in last view
	chs.ViewTimer.Stop()

	// get the message with the hignest QC, and update local genericQC
	hignQcMsg, _ := GetHighChainedQC(&chs.NewViewMsgs)
	chs.GenericQC = hignQcMsg.Justify

	// create a new hotstuff node extend the hignest QC's node and local HsNode
	if len(chs.CurProposal.Commands) == 0 {
		chs.BlkStore.GenEmptyBlock()
		chs.BlkStore.Height += 1
	} else {
		reqs := common.CutOffTwoDimByteSlice(chs.CurProposal.Commands, 128)
		chs.BlkStore.GenNewBlock(chs.View.ViewNumber, common.TwoDimByteSlice2StringSlice(reqs), chs.BlkStore.GeneratedHeight)
	}

	// leader create leaf node extend from the hignest QC's node but doesn't update it to local CHsNode
	// leader update block to lock block
	newCHsNode := chs.CreateLeaf((hignQcMsg.Justify.HsNodes[0]).CurHash)
	msgCHsNode := [4]common.HsNode{newCHsNode, chs.HsNodes[0], chs.HsNodes[1], chs.HsNodes[2]}
	chs.UpdateBlock(&chs.BlkStore.CurProposalBlk)

	// update local state
	chs.CurPhase = hstypes.HALF_GENERIC
	chs.LastLeaderState = false
	chs.ExecuteState = false
	chs.NewViewMsgs = make([]*hstypes.CMsg, 0)
	chs.GenericVoteMsgs = make([]*hstypes.CMsg, 0)

	// log
	// chs.Logger.Println("[NEW_VIEW]", chs.GetNodeName(), " ViewNumber:", chs.View.ViewNumber, " Success!", chs.CurProposal.Commands[0][:2])
	return &hstypes.CMsg{
		MType:      hstypes.GENERIC,
		ViewNumber: chs.View.ViewNumber,
		SendNode:   chs.GetNodeName(),
		ReciNode:   "Broadcast",
		HsNodes:    msgCHsNode,
		Proposal:   chs.CurProposal,
		Blk:        chs.BlkStore.CurProposalBlk,
		Justify:    chs.GenericQC,
	}
}

// CHandleGeneric: the replica in generic phase handle the message
// CHandleGeneric implement chained hotstuff description as follow:
// as a replica
// wait for message m from leader(curView)
//
//	m : matchingMsg(m, generic, curView)
//
// b∗ ←m.node; b′′ ← b∗.justify.node;
// b′ ← b′′.justify.node; b ← b′.justify.node;
//
// if safeNode(b∗, b∗.justify) then
//
//	send voteMsg(generic, b∗, ⊥) to leader(curView)
//
// if b∗ .parent = b′′ then
//
//	genericQC ← b∗ .justify
//
// if (b∗ .parent = b′′) ∧ (b′′.parent = b′)
//
//	then lockedQC ← b′′.justify
//
// if (b∗ .parent = b′′) ∧ (b′′.parent = b′) ∧  (b′.parent = b) then
//
//	execute new commands through b, respond to clients
func (chs *CHotstuff) CHandleGeneric(msg *hstypes.CMsg) []*hstypes.CMsg {

	// check whether message is from this view
	if msg.ViewNumber != chs.View.ViewNumber {
		// fmt.Println(msg.ViewNumber, chs.View.ViewNumber)
		return nil
	}

	// check whether message's nodes and it's new node is safe
	if !chs.CheckNewCHsNode(msg) {
		return nil
	}

	chs.ViewChangeFlag = false

	// if this node isn't leader, update the proposal and node carried by the message
	if !chs.IsLeader() {

		chs.UpdateBlock(&msg.Blk)

		// stop view timer set in the previous "CHandleGeneric" func in last view
		chs.ViewTimer.Stop()
	}

	// update local HsNodes
	chs.HsNodes = msg.HsNodes

	// sign for message
	partSig, err := chs.ThresholdSigner.ThresholdSign(msg.ChainedMessage2Byte())
	if err != nil {
		return nil
	}

	// generate generic-vote message
	genericVote := &hstypes.CMsg{
		MType:      hstypes.GENERIC_VOTE,
		ViewNumber: msg.ViewNumber,
		ReciNode:   chs.GetChainedCurLeader(),
		HsNodes:    msg.HsNodes,
		PartialSig: partSig,
	}

	// b*.parent = b"
	if bytes.Equal(msg.HsNodes[0].ParentHash, msg.HsNodes[1].CurHash) {
		chs.GenericQC = msg.Justify

		// b*.parent = b" && b".parent = b'
		if bytes.Equal(msg.HsNodes[1].ParentHash, msg.HsNodes[2].CurHash) {
			chs.LockedQC = msg.Justify

			// b*.parent = b" && b".parent = b' && b'.parent = b
			if bytes.Equal(msg.HsNodes[2].ParentHash, msg.HsNodes[3].CurHash) {
				chs.ExecuteState = true

				// if block[3] isn't empty and its hash and lock HsNodes[3].CurHash is equal, store the block
				if !chs.Blocks[3].IsEmpty() && bytes.Equal(chs.Blocks[3].Hash(), chs.HsNodes[3].CurHash) {
					// fmt.Println("store block",
					// 	common.String2ByteSlice(chs.Blocks[0].BlkData.Trans)[0][:5],
					// 	common.String2ByteSlice(chs.Blocks[1].BlkData.Trans)[0][:5],
					// 	common.String2ByteSlice(chs.Blocks[2].BlkData.Trans)[0][:5],
					// 	common.String2ByteSlice(chs.Blocks[3].BlkData.Trans)[0][:5])
					chs.Blocks[3].BlkHdr.Validation = msg.Justify.Sign
					// chs.BlkStore.CurProposalBlk.BlkHdr.Validation = msg.Justify.Sign
					chs.BlkStore.CurBlkHash = chs.Blocks[3].Hash()
					chs.BlkStore.StoreBlock(chs.Blocks[3])
					chs.BlkStore.Height -= 1
				}
			}
		}
	}

	// log
	// chs.Logger.Println("[GENERIC]", "r_"+strconv.Itoa(chs.ConsId)+" ViewNumber:"+strconv.Itoa(chs.View.ViewNumber)+" Success!")

	chs.NewViewMsgs = make([]*hstypes.CMsg, 0)
	// if this node is not leader, send new view message
	if !chs.IsLeader() {

		chs.View.NextView()
		newViewMsg := &hstypes.CMsg{
			MType:      hstypes.NEW_VIEW,
			ViewNumber: chs.View.ViewNumber,
			HsNodes:    chs.HsNodes,
			Justify:    chs.GenericQC,
		}
		newViewMsg.ReciNode = chs.GetChainedCurLeader()

		// init the the flag of recieved last leader state
		if chs.GetNodeName() == chs.GetLeaderName() {
			chs.LastLeaderState = false
		}

		// set the timer to ensure that this node succeed entering the next view and propose a new proposal
		chs.ViewTimer.Start(func() {
			// chs.Logger.Println("[TIMER-EXPIRE]:", chs.GetNodeName(), "generic")
			chs.StartViewChange()
		}, func() {
			// chs.Logger.Println("generic View Timer stop", chs.GetNodeName(), chs.View.ViewNumber)
		})

		return []*hstypes.CMsg{genericVote, newViewMsg}
	}
	return []*hstypes.CMsg{genericVote}
}

// CHandleGenericVote: the leader in leader-half handle the message
// CHandleGenericVote implement chained hotstuff description as follow:
// as a leader // pre-commit phase (leader-half)
// wait for (n − f) votes:
//
//	V ← {v | matchingMsg(v, generic, curView)}
//
// genericQC ← QC(V)
func (chs *CHotstuff) CHandleGenericVote(msg *hstypes.CMsg) []*hstypes.CMsg {

	// check the node whether in leader-half phase, and waiting for votes
	if chs.CurPhase != hstypes.HALF_GENERIC {
		return nil
	}

	// check whether message's type and view number are matching current view
	if CheckCMsg(msg, hstypes.GENERIC_VOTE, chs.View.ViewNumber) {
		chs.GenericVoteMsgs = append(chs.GenericVoteMsgs, msg)
	}

	// check meet the threshold conditions, (m > 2f+1)
	if len(chs.GenericVoteMsgs) <= (chs.View.NodesNum-1)/3*2 {
		return nil
	}

	// stop the view timer set in the previous "CHandleNewView" func in last view
	chs.ViewTimer.Stop()

	// generate genericQC, and combine recieved part signature to a complete signature
	// the add it to genericQC
	genericQC := hstypes.ChainedQC{
		QType:      hstypes.GENERIC,
		ViewNumber: msg.ViewNumber,
		HsNodes:    msg.HsNodes,
	}
	genericSig := chs.CombineSign(chs.GenericVoteMsgs)
	if genericSig == nil {
		return nil
	}
	genericQC.Sign = genericSig

	// log
	// chs.Logger.Println("[HALF_GENERIC]", "r_"+strconv.Itoa(chs.ConsId)+" ViewNumber:"+strconv.Itoa(chs.View.ViewNumber)+" Success!")

	// update leader's local GenericQC, view and phase state
	chs.GenericQC = genericQC
	chs.View.NextView()
	chs.CurPhase = hstypes.NEW_VIEW
	chs.CurProposal = hstypes.Proposal{}

	// set the view timer to ensure that the node succeed in next view
	// this timer works the same as the duplicate timer set in the "CHandleGeneric" function
	// the only difference is that this is where the leader is set
	chs.ViewTimer.Start(func() {
		// chs.Logger.Println("[TIMER-EXPIRE]:", chs.GetNodeName(), "collect votes")
		chs.StartViewChange()
	}, func() {
		// chs.Logger.Println("handle vote View Timer stop", chs.GetNodeName(), chs.View.ViewNumber)
	})

	// return the new-view message and unicast it to the next leader of the next view
	return []*hstypes.CMsg{{
		MType:      hstypes.NEW_VIEW,
		HsNodes:    chs.HsNodes,
		ReciNode:   chs.GetChainedCurLeader(),
		ViewNumber: chs.View.ViewNumber,
		Justify:    chs.GenericQC,
	},
	}
}

// GetHighChainedQC: in chained-hotstuff, get the highest QC
// params: recieved n-f new-view messages
// return: the point of message with the highest QC
func GetHighChainedQC(newViewMsgs *[]*hstypes.CMsg) (*hstypes.CMsg, error) {
	msgNum := len(*newViewMsgs)
	maxIndex := 0
	maxViewNumber := (*newViewMsgs)[0].Justify.ViewNumber
	for i := 0; i < msgNum; i++ {
		if (*newViewMsgs)[i].Justify.ViewNumber > maxViewNumber {
			maxIndex = i
			maxViewNumber = (*newViewMsgs)[i].Justify.ViewNumber
		}
		// fmt.Println("GetHighChainedQC", (*newViewMsgs)[i].SendNode, (*newViewMsgs)[i].HsNodes, (*newViewMsgs)[i].Justify.ViewNumber)
	}
	return (*newViewMsgs)[maxIndex], nil
}

// CheckNewHsNode: check whether chained message.node is valid
// params: chained message
// return: a boolean
func (chs *CHotstuff) CheckNewCHsNode(msg *hstypes.CMsg) bool {
	if msg.HsNodes[0].ParentHash == nil || msg.HsNodes[1].CurHash == nil {
		chs.Logger.Println("nil err", chs.ConsId, msg.HsNodes[0].ParentHash == nil, msg.HsNodes[1].CurHash == nil)
		return false
	}
	if !bytes.Equal(msg.HsNodes[0].ParentHash, msg.HsNodes[1].CurHash) {
		chs.Logger.Println("bytes err", chs.GetNodeName(), chs.CurPhase, chs.View.ViewNumber, msg.HsNodes[0].ParentHash, msg.HsNodes[1].CurHash)

		return false
	}
	if !chs.SafeNode(msg.HsNodes, &msg.Justify) {
		chs.Logger.Println("safe err", chs.SafeNode(msg.HsNodes, &msg.Justify))
		return false
	}
	return true
}

// StartViewChange: start the view-change protocol when the timer expire
func (chs *CHotstuff) StartViewChange() {
	// update the view number to expect to enter
	chs.View.NextView()
	chs.CurPhase = hstypes.NEW_VIEW
	chs.ViewChangeFlag = true

	// generate new-view message
	newViewMsg := hstypes.CMsg{
		MType:      hstypes.NEW_VIEW,
		Justify:    chs.GenericQC,
		ViewNumber: chs.View.ViewNumber,
		SendNode:   chs.GetNodeName(),
		ReciNode:   chs.GetLeaderName(),
	}
	sign, err := chs.ThresholdSigner.ThresholdSign(newViewMsg.ChainedMessage2Byte())
	if err == nil {
		newViewMsg.PartialSig = sign
	}

	// send the message
	newViewMsgJson, err := json.Marshal(newViewMsg)
	if err == nil {
		if chs.ForwardChan != nil {

			chs.ViewChangeSendFlag = true
			chs.ForwardChan <- newViewMsgJson
		}
		if chs.SendChan != nil {
			serMsg := message.ServerMsg{
				SType:      message.ORDER,
				SendServer: newViewMsg.SendNode,
				ReciServer: newViewMsg.ReciNode,
				Payload:    newViewMsgJson,
			}
			chs.SendChan <- serMsg
		}
	}

	// log
	chs.Logger.Println("[VIEW_CHANGE]:", chs.GetNodeName(), "ViewNumber:", chs.View.ViewNumber)

	// set the view timer to ensure to enter the new view correctly for liveness
	chs.ViewTimer.Start(func() {
		// chs.Logger.Println("StartViewChange View Timer stop", chs.GetNodeName())
		chs.StartViewChange()
	}, func() {
		// chs.Logger.Println("StartViewChange View Timer stop", chs.GetNodeName())
	})
}

// GetNodeName: return self node name
func (chs *CHotstuff) GetNodeName() string {
	return "r_" + strconv.Itoa(chs.ConsId)
}

// GetLeaderName:return leader of current view name
func (chs *CHotstuff) GetLeaderName() string {
	return chs.View.LeaderName()
}
