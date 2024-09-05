package bcmanager

import (
	"mgmt"
)

// NodeManager: the basic node manager extend the orignal node manager
type NodeManager struct {
	mgmt.NodeManager                     // extend the orignal node manager
	SyncMsgs         []*mgmt.NodeMgmtMsg //the slice of recieved sync message
}

// NewNodeManager: generate a new node manager
// params:
// id: 				the unique identification of the server
// nodesTable: 		the all known node information table in system
// nodesChannel: 	the channels table of all nodes
func NewNodeManager(id int, nodesTable map[string]mgmt.NodeKey, nodesChannel map[string]chan []byte) *NodeManager {
	newNodeManager := &NodeManager{
		*mgmt.NewNodeManager(id, nodesTable, nodesChannel),
		make([]*mgmt.NodeMgmtMsg, 0),
	}

	return newNodeManager
}
