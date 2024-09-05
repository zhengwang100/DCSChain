package server

import (
	"encoding/json"
	"message"
	"mgmt"
	"time"
)

func (s *Server) StartNodeExit() {

	switch s.NMType {
	case mgmt.BASIC:
		s.StartBCNodeExit()
	}
}

// StartBCNodeExit: start node exit the system
func (s *Server) StartBCNodeExit() {
	// s.Orderer.BasicHotstuff.Logger.Println("StartNodeExit", s.ServerID.ID.Name)

	// update the local node manager mode to EXIT
	s.NodeManager.Mode = mgmt.EXIT

	for _, nodeKey := range s.NodeManager.NodesTable {
		// fmt.Println(nodeKey.Name)
		if nodeKey.Name == s.ServerID.ID.Name {
			// node which exits only stop
			s.StopOrderer()
			continue
		}

		// generate the exit message
		exitMsg := mgmt.NodeMgmtMsg{
			Type:     mgmt.EXIT,
			NMType:   mgmt.NM_APPLY,
			Leader:   int(time.Now().UnixNano()) / int(time.Millisecond),
			SendNode: s.ServerID.ID.Name,
			ReciNode: nodeKey.Name,
		}

		// send exit message
		msgJson, err := json.Marshal(exitMsg)
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
