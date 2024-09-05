package core

import (
	"bytes"
	"common"
	"fmt"
	hstypes "hotstuff/types"
)

// HandleNewView: the leader in prepare phase handle the message
// HandleNewView implement basic hotstuff description as follow:
// as a leader:
// wait for (n − f) new-view messages:
//
//	M ← {m | matchingMsg(m, new-view, curView−1)}
//
// highQC ← (arg max {m.justify.viewNumber}).justify
// curProposal ← createLeaf(highQC.node, client’s command)
// broadcast Msg(prepare, curProposal, highQC)
func (bhs *BCHotstuff) HandleNewView(msg *hstypes.Msg) *hstypes.Msg {
	bhs.ProposalLock.Lock()
	defer bhs.ProposalLock.Unlock()

	// check the node whether in new-view phase
	if bhs.CurPhase != hstypes.NEW_VIEW && bhs.CurPhase != hstypes.WAITING {
		return nil
	}

	// check whether this node is leader
	if bhs.GetNodeName() != bhs.GetLeaderName() {
		return nil
	}

	// check whether message's type and view number are matching current view
	if len(bhs.LastRoundMsg) == 0 {
		if bhs.MatchingMsg(msg, hstypes.NEW_VIEW, bhs.View.ViewNumber) && bhs.CheckQC(msg, hstypes.PREPARE, bhs.View.ViewNumber-1) {
			bhs.NewViewMsgs = append(bhs.NewViewMsgs, msg)
		}
	} else if bhs.MatchingMsg(msg, hstypes.NEW_VIEW, bhs.View.ViewNumber) && bhs.CheckQC(msg, hstypes.PREPARE, bhs.LastRoundMsg[len(bhs.LastRoundMsg)-1].ViewNumber) {
		bhs.NewViewMsgs = append(bhs.NewViewMsgs, msg)
	}

	// check meet the threshold conditions, (m > 2f+1)
	if len(bhs.NewViewMsgs) <= (bhs.View.NodesNum-1)/3*2 {
		return nil
	}
	// we assume the requests always are sent to the leader or the next node of the leader
	// so, if the current proposal is empty, then wait for a valid request or proposal
	if len(bhs.CurProposal.Commands) == 0 {
		if bhs.ViewChangeSendFlag {
			return nil
		}
		bhs.CurPhase = hstypes.WAITING
		return nil
	}

	// stop view timer, enter new view successfully
	bhs.ViewTimer.Stop()

	// create new node with new command extend from last node
	// get the index for message with the highest QC from n-f new-view message
	// change the node local state to prepare
	// reqs := common.CutOffTwoDimByteSlice(bhs.CurProposal.Command, 128)
	bhs.BlkStore.GenNewBlock(bhs.View.ViewNumber, common.TwoDimByteSlice2StringSlice(bhs.CurProposal.Commands))
	HignQCNum := GetHighQCIndex(&bhs.NewViewMsgs)
	bhs.HsNode = bhs.CreateLeaf(bhs.NewViewMsgs[HignQCNum].Justify.HsNode.CurHash)
	// fmt.Println("gener proposal Handle NewView", bhs.GetNodeName(), bhs.View.ViewNumber, bhs.CurPhase, bhs.HsNode)
	bhs.CurPhase = hstypes.PREPARE

	// log
	// bhs.Logger.Println("New Round in view", bhs.View.ViewNumber, ":"+strconv.Itoa(bhs.View.ViewNumber))
	// bhs.Logger.Println("[NEW_VIEW]:", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber, len(bhs.BlkStore.CurProposalBlk.BlkData.Trans))

	bhs.ViewTimer.Start(func() {
		// bhs.Logger.Println("newview View Timer stop", bhs.GetNodeName())
		bhs.StartViewChange()
	}, func() {
		// bhs.Logger.Println("newview View Timer stop", bhs.GetNodeName())
	})

	// return the prepare message from leader
	// then the message will be broadcast to all replica
	return &hstypes.Msg{
		MType:      hstypes.PREPARE,
		ViewNumber: bhs.View.ViewNumber,
		HsNode:     bhs.HsNode,
		Justify:    bhs.NewViewMsgs[HignQCNum].Justify,
		Proposal:   bhs.CurProposal,
		Block:      bhs.BlkStore.CurProposalBlk,
		ReciNode:   "Broadcast",
	}
}

// GetHighQCIndex: from some messages, return the index of message with the highest QC
// params: recieved new-view messages
// return: the index
func GetHighQCIndex(newViewMsgs *[]*hstypes.Msg) int {
	msgNum := len(*newViewMsgs)
	maxIndex := 0
	maxViewNumber := (*newViewMsgs)[0].Justify.ViewNumber
	for i := 0; i < msgNum; i++ {
		// fmt.Println((*newViewMsgs)[i].SendNode, (*newViewMsgs)[i].Justify.ViewNumber)
		if (*newViewMsgs)[i].Justify.ViewNumber > maxViewNumber {
			maxIndex = i
			maxViewNumber = (*newViewMsgs)[i].Justify.ViewNumber
		}
	}
	return maxIndex
}

// CheckNewHsNode: check whether message.node is valid
// params: message
// return: a boolean
func (bhs *BCHotstuff) CheckNewHsNode(prepareMsg *hstypes.Msg) bool {
	// check whether node is nil
	if prepareMsg.HsNode.ParentHash == nil || prepareMsg.Justify.HsNode.CurHash == nil {
		fmt.Println("nil err")
		return false
	}

	// this message'node extend from message's Justify node
	if !bytes.Equal(prepareMsg.HsNode.ParentHash, prepareMsg.Justify.HsNode.CurHash) {
		fmt.Println("bytes err", prepareMsg.HsNode.ParentHash, prepareMsg.Justify.HsNode.CurHash)
		return false
	}

	// check whether message's node is safe node
	if !bhs.SafeNode(&prepareMsg.HsNode, &prepareMsg.Justify) {
		fmt.Println("not safe node")
		return false
	}
	return true
}

// GenProposal: leader generates a new proposal and processes prepare phase
// return:
// - the prepare message
func (bhs *BCHotstuff) GenProposal() *hstypes.Msg {
	// fmt.Println(bhs.GetNodeName(), bhs.View.ViewNumber, len(bhs.NewViewMsgs), (bhs.View.NodesNum-1)/3*2)
	// check meet the threshold conditions, (m > 2f+1)

	// fmt.Println(len(bhs.NewViewMsgs), (bhs.View.NodesNum-1)/3*2)
	if len(bhs.NewViewMsgs) <= (bhs.View.NodesNum-1)/3*2 && !bhs.IgnoreCheckQC {
		return nil
	}
	bhs.ViewTimer.Stop()
	bhs.CurPhase = hstypes.PREPARE

	// create new node with new command extend from last node
	// get the index for message with the highest QC from n-f new-view message
	// change the node local state to prepare
	reqs := common.CutOffTwoDimByteSlice(bhs.CurProposal.Commands, 128)
	bhs.BlkStore.GenNewBlock(bhs.View.ViewNumber, common.TwoDimByteSlice2StringSlice(reqs))
	HignQCNum := GetHighQCIndex(&bhs.NewViewMsgs)
	bhs.HsNode = bhs.CreateLeaf(bhs.NewViewMsgs[HignQCNum].Justify.HsNode.CurHash)
	// fmt.Println("gener proposal ", bhs.GetNodeName(), bhs.View.ViewNumber, bhs.HsNode)

	bhs.IgnoreCheckQC = false

	// log
	// bhs.Logger.Println("New Round in view", bhs.View.ViewNumber, ":"+strconv.Itoa(bhs.View.ViewNumber))
	// bhs.Logger.Println("[NEW_VIEW]:", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber, len(bhs.BlkStore.CurProposalBlk.BlkData.Trans))

	bhs.ViewTimer.Start(func() {
		// bhs.Logger.Println("newview View Timer stop", bhs.GetNodeName())
		bhs.StartViewChange()
	}, func() {
		// bhs.Logger.Println("newview View Timer stop", bhs.GetNodeName())
	})

	// return the prepare message from leader
	// then the message will be broadcast to all replica
	return &hstypes.Msg{
		MType:      hstypes.PREPARE,
		ViewNumber: bhs.View.ViewNumber,
		HsNode:     bhs.HsNode,
		Justify:    bhs.NewViewMsgs[HignQCNum].Justify,
		Proposal:   bhs.CurProposal,
		Block:      bhs.BlkStore.CurProposalBlk,
		SendNode:   bhs.GetNodeName(),
		ReciNode:   "Broadcast",
	}
}
