package server

import (
	"encoding/json"
	"message"
	"mgmt"
)

// StartNodeJoin: add nodes to the system according to different rules
func (s *Server) StartNodeJoin(simulateServers []*Server) {
	s.Logger.Println("Start Node Join", len(s.NodeManager.NodesTable))

	switch s.NMType {
	case mgmt.BASIC:
		s.StartBCNodeJoin(simulateServers)
	}
}

// StartBCNodeJoin: start a new node join the system by basic method, simulate new node get all orignal node information in system
// params:
// - simulateServers: the node in system
func (s *Server) StartBCNodeJoin(simulateServers []*Server) {
	// set the node manager mode to JOIN
	s.NodeManager.Mode = mgmt.JOIN
	for _, node := range simulateServers {
		node.NodeManager.NewNode.Chan = s.NodeManager.NewNode.Chan
	}

	// simulate new node get all orignal node information in system
	for _, nodeKey := range s.NodeManager.NodesTable {
		// fmt.Println(nodeKey.Name)
		if nodeKey.Name == s.ServerID.ID.Name {
			continue
		}

		nKey := mgmt.NodeKey{
			Name:      s.NodeManager.NewNode.Name,
			Sm2PubKey: s.ServerID.ID.PubKey,
			Sm4Key:    nodeKey.Sm4Key,
		}
		joinMsg := mgmt.NodeMgmtMsg{
			Type:     mgmt.JOIN,
			NMType:   mgmt.NM_APPLY,
			NodeKey:  nKey,
			SendNode: s.ServerID.ID.Name,
			ReciNode: nodeKey.Name,
		}

		// send join message
		msgJson, err := json.Marshal(joinMsg)
		if err == nil {
			go s.SendMsg(message.ServerMsg{
				SType:      message.NODEMGMT,
				SendServer: s.ServerID.ID.Name,
				ReciServer: nodeKey.Name,
				Payload:    msgJson,
			})
		}
	}

	// update local node manager state
	s.NodeManager.State = mgmt.NM_APPLY
}
