package server

// UpdateNodeInfo: server update node manager and orderer
// params:
// msg: the message with sync information
func (s *Server) UpdateNodeInfo(index int) {
	if s.ServerID.ID.Name != s.NodeManager.NewNode.Name {
		// the original nodes update node-manager and consensus information

		// update information
		s.NodeManager.UpdateNewNodeInfo()
		s.Orderer.UpdateNodesNum(len(s.NodeManager.NodesTable))

		// reset the node-manager state
		s.NodeManager.ResetNodeManager()
	} else {
		// the new node update itself
		s.Orderer.UpdateNodesNum(len(s.NodeManager.NodesTable))

		s.Orderer.SyncInfo(s.NodeManager.SyncMsgs[index], s.NodeManager.GetLeaderFromSyncMsgs(s.NodeManager.SyncMsgs))
		s.NodeManager.ResetNodeManager()
	}
}

// StopOrderer: stop the orderer
func (s *Server) StopOrderer() {
	s.Orderer.Stop()
}

// RestartOrderer: reset the orderer's state and restart the orderer
func (s *Server) RestartOrderer() {
	s.Orderer.ResetState()
	go s.HandleReq()
	s.Orderer.RestartCons()
}
