package mgmt

import (
	"log"
	"os"
	"strconv"
	"sync"
)

// NodeManager: the most orignal nodemanager keeping the system running, other methods of Nodemanagers are inherited from it
type NodeManager struct {
	mu      sync.Mutex      // exclusive lock
	NMID    int             // unique identifier of node manager, the value must be the same as that of a node
	NewNode NodeInfo        // the new node information
	State   StateType       // the state of the join or exit process
	Mode    NodeManagerMode // the mode of node manager, include INACTIVE/JOIN/EXIT

	Logger       log.Logger             `json:"logger"`       // the logger
	NodesTable   map[string]NodeKey     `json:"NodesTable"`   // the all known node PubKey table in system
	NodesChannel map[string]chan []byte `json:"NodesChannel"` // the all known node channel table in system
}

// NewNodeManager: generate a new node manager
// params:
// id: 				the unique identification of the server
// nodesTable: 		the all known node information table in system
// nodesChannel: 	the channels table of all nodes
func NewNodeManager(id int, nodesTable map[string]NodeKey, nodesChannel map[string]chan []byte) *NodeManager {
	newNodeManager := &NodeManager{
		NMID:         id,
		NewNode:      NodeInfo{},
		State:        NM_INACTIVE,
		Mode:         0,
		mu:           sync.Mutex{},
		NodesTable:   nodesTable,
		NodesChannel: nodesChannel,
		Logger:       *log.New(os.Stdout, "", 0),
	}

	newNodeManager.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	return newNodeManager
}

// UpdateSm4Key: add a name-key to node manager
// params:
// name: the node identity
// key: the name corresponding key
func (nm *NodeManager) UpdateSm4Key(name string, key []byte) {
	if _, ok := nm.NodesTable[name]; !ok {
		// if no, add a key-value pair
		nm.NodesTable[name] = NodeKey{
			Name:   name,
			Sm4Key: key,
		}
	} else {
		nk := nm.NodesTable[name]
		nk.Sm4Key = key
		nm.NodesTable[name] = nk
	}
}

// GetName: get the name of node manager
func (nm *NodeManager) GetName() string {
	return "r_" + strconv.Itoa(nm.NMID)
}

// update the new node information in local node manager
func (nm *NodeManager) UpdateNewNodeInfo() {
	if nm.Mode == JOIN {
		nm.NodesChannel[nm.NewNode.Name] = nm.NewNode.Chan
		nm.NodesTable[nm.NewNode.Name] = nm.NewNode.NodeKey
	} else if nm.Mode == EXIT {
		delete(nm.NodesChannel, nm.NewNode.Name)
		delete(nm.NodesTable, nm.NewNode.Name)
	}
}

// ResetNodeManager: reset new node info in node manager
func (nm *NodeManager) ResetNodeManager() {
	nm.NewNode = NodeInfo{}
	nm.State = NM_INACTIVE
	nm.Mode = 0
}

// GetNodeNames: get the all nodes' name in node mananger
func (nm *NodeManager) GetNodeNames() []string {
	keys := make([]string, 0, len(nm.NodesChannel))
	for k := range nm.NodesChannel {
		keys = append(keys, k)
	}
	return keys
}

// GetOtherNodeNames: get the all nodes' name in node mananger except self
func (nm *NodeManager) GetOtherNodeNames() []string {
	name := "r_" + strconv.Itoa(nm.NMID)
	keys := make([]string, 0, len(nm.NodesChannel)-1)
	for k := range nm.NodesChannel {
		if k == name {
			continue
		}
		keys = append(keys, k)
	}
	return keys
}
