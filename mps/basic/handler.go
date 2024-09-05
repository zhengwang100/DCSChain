package bcmanager

import (
	"mgmt"
)

// HanleJoin: the nodes in original system handle the join message
func (nm *NodeManager) HandleJoin(msg *mgmt.NodeMgmtMsg) *mgmt.NodeMgmtMsg {
	// update new node to local node manager
	nm.NewNode.Name = msg.SendNode
	nm.NewNode.NodeKey = msg.NodeKey

	// update local mode
	nm.Mode = mgmt.JOIN

	// log
	nm.Logger.Println("[JOIN]:", nm.GetName(), "ready for join node:", nm.NewNode.Name)

	// send sync message
	return &mgmt.NodeMgmtMsg{
		Type:     mgmt.JOIN,
		NMType:   mgmt.NM_SYNC,
		ReciNode: nm.NewNode.Name,
	}
}

// HandleSync: the new node handle the sync message to reach the state which is the same as other nodes in system
func (nm *NodeManager) HandleSync(msg *mgmt.NodeMgmtMsg) (int, *mgmt.NodeMgmtMsg) {

	// log the sync message
	nm.SyncMsgs = append(nm.SyncMsgs, msg)

	// check the threshold
	if len(nm.SyncMsgs) <= (len(nm.NodesTable)-1)/3*2 {
		return -1, nil
	}

	// if the new node has sync successfully or is inactive, stop here and return nil
	if nm.State == mgmt.NM_SYNC || nm.State == mgmt.NM_INACTIVE {
		return -1, nil
	}

	// select the index message with the heightest qc, and update local state
	HignQCNum := GetHighQCIndex(nm.SyncMsgs)
	nm.State = mgmt.NM_SYNC

	// log
	nm.Logger.Println("[SYNC]:", nm.GetName(), "sync succeed")

	// return the message that should have the highest qc for synchronization
	// and sends the restart message to the node in the original system
	return HignQCNum, &mgmt.NodeMgmtMsg{
		Type:     mgmt.JOIN,
		NMType:   mgmt.NM_RESTART,
		SendNode: nm.NewNode.Name,
		ReciNode: "Gossip",
	}
}

// HandleExit: the nodes in original system handle the exit message
func (nm *NodeManager) HandleExit(msg *mgmt.NodeMgmtMsg) *mgmt.NodeMgmtMsg {
	// update new node to local node manager
	nm.NewNode.Name = msg.SendNode
	nm.NewNode.Chan = nm.NodesChannel[nm.NewNode.Name]

	// update local mode
	nm.Mode = mgmt.EXIT

	// log
	nm.Logger.Println("[EXIT]:", nm.GetName(), "ready for exit node:", nm.NewNode.Name)

	// send sync message
	return &mgmt.NodeMgmtMsg{
		Type:     mgmt.EXIT,
		NMType:   mgmt.NM_AGREE,
		SendNode: nm.GetName(),
		ReciNode: nm.NewNode.Name,
	}
}

// HandleAgree: the nodes in original system handle the agree message
func (nm *NodeManager) HandleAgree(msg *mgmt.NodeMgmtMsg) *mgmt.NodeMgmtMsg {
	// log the sync message
	nm.SyncMsgs = append(nm.SyncMsgs, msg)

	// check the threshold
	if len(nm.SyncMsgs) <= (len(nm.NodesTable)-1)/3*2 {
		return nil
	}

	// if the new node has sync successfully or is inactive, stop here and return nil
	if nm.State == mgmt.NM_AGREE || nm.State == mgmt.NM_INACTIVE {
		return nil
	}
	nm.State = mgmt.NM_AGREE

	// log
	nm.Logger.Println("[AGREE]:", nm.GetName(), "exit succeed")

	// send restart message
	return &mgmt.NodeMgmtMsg{
		Type:     mgmt.EXIT,
		NMType:   mgmt.NM_RESTART,
		SendNode: nm.GetName(),
		ReciNode: "Gossip",
	}
}

// GetHighQCIndex: from some messages, return the index of message with the highest QC
func GetHighQCIndex(syncMsgs []*mgmt.NodeMgmtMsg) int {
	msgNum := len(syncMsgs)
	maxIndex := 0
	maxViewNumber := syncMsgs[0].Justify.ViewNumber
	for i := 0; i < msgNum; i++ {
		if syncMsgs[i].Justify.ViewNumber > maxViewNumber {
			maxIndex = i
			maxViewNumber = syncMsgs[i].ViewNumber
		}
	}
	return maxIndex
}

// GetLeaderFromSyncMsgs: select the leader from 2f+1 sync message
func (nm *NodeManager) GetLeaderFromSyncMsgs(syncMsgs []*mgmt.NodeMgmtMsg) int {
	leaders := make(map[int]int, 0)

	for _, m := range syncMsgs {
		leaders[m.Leader] += 1
	}
	for l, count := range leaders {
		if count > (len(nm.NodesTable)-1)/3*2 {
			return l
		}
	}
	return -1
}
