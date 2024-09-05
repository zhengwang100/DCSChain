package server

import (
	"bcrequest"
	"encoding/json"
	"message"
	"mgmt"
	"time"
)

// HandleReq: the node recieve request and submit or transmit it
func (s *Server) HandleReq() {
	for s.Orderer.ReqState {

		// recieve the flag of submitting the request in a blocking manner
		<-s.Orderer.ReqFlagChan

		// this lock is required instead of TryLock because a commit must be made to maintain the viability of the system when a commit request message is received.
		s.RequestsLock.Lock()

		// in the basic hotstuff, only the leader put forward a proposal
		if len(s.Requests) != 0 && len(s.Requests) <= s.BatchSize && s.ServerID.ID.Name == s.Orderer.GetLeaderName() {
			if s.VerifyReqs() {
				// fmt.Println("Verify succeed", s.ServerID.ID.Name, s.Orderer.BasicHotstuff.View.ViewNumber)
				s.Orderer.HandleReq(s.BlkStore.Height, s.BlkStore.PreBlkHash, s.BlkStore.CurBlkHash, s.Requests)
				s.Requests = make([]bcrequest.BCRequest, 0)
			} else {
				// fmt.Println("Verify failed", s.ServerID.ID.Name, s.Orderer.BasicHotstuff.View.ViewNumber)
				s.Requests = make([]bcrequest.BCRequest, 0)
			}
		}
		s.RequestsLock.Unlock()
	}
}

// HandleNodeManagerMsg: handle the message to node manager
// params:
// - payload: the payload of the server message is the encoded ndoe-manager message
func (s *Server) HandleNodeManagerMsg(payload []byte) {
	switch s.NMType {
	case mgmt.BASIC:

		// decode the message
		msg := &mgmt.NodeMgmtMsg{}
		err := json.Unmarshal(payload, msg)
		if err != nil {
			return
		}

		// handle the message
		if msg.Type == mgmt.JOIN {
			if msg.NMType == mgmt.NM_APPLY {

				// the original node stop and send sync message
				s.StopOrderer()
				msgReturn := s.NodeManager.HandleJoin(msg)

				// update orderer and node manager
				s.Orderer.AddSyncInfo(msgReturn)
				// fmt.Println(msgReturn.Block[0].BlkData.Trans)
				s.UpdateNodeInfo(-1)

				// send join message
				msgJson, err := json.Marshal(msgReturn)
				if err == nil {
					go s.SendMsg(message.ServerMsg{
						SType:      message.NODEMGMT,
						SendServer: s.ServerID.ID.Name,
						ReciServer: msgReturn.ReciNode,
						Payload:    msgJson,
					})
				}
			} else if msg.NMType == mgmt.NM_SYNC {

				// if recieve sync message, the new node sync
				msgSync, msgReturn := s.NodeManager.HandleSync(msg)
				if msgSync != -1 && msgReturn != nil && s.NodeManager.State == mgmt.NM_SYNC {

					s.UpdateNodeInfo(msgSync)

					// send join message
					msgJson, err := json.Marshal(msgReturn)
					if err == nil {
						go s.SendMsg(message.ServerMsg{
							SType:      message.NODEMGMT,
							SendServer: s.ServerID.ID.Name,
							ReciServer: msgReturn.ReciNode,
							Payload:    msgJson,
						})
					}
				}
			} else if msg.NMType == mgmt.NM_RESTART {

				// recieve the RESTART message, restart the orderer after waiting the signers update
				for !s.Orderer.IsReady() {

					// for s.Orderer.BasicHotstuff.View.NodesNum!= s.Orderer.BasicHotstuff.ThresholdSigner.SignNum{
					time.Sleep(10 * time.Millisecond)
				}
				s.RestartOrderer()

				// if the new node causes the threshold f to change, add an additional empty new view message
				if s.ServerID.ID.Name == s.Orderer.GetLeaderName() {
					// n.BasicHotstuff.IgnoreCheckQC = true

					// refresh the leader
					s.Orderer.FixLeader()
				}
			}
		} else if msg.Type == mgmt.EXIT {
			if msg.NMType == mgmt.NM_APPLY {

				// the original node stop and send agree message
				s.StopOrderer()

				msgReturn := s.NodeManager.HandleExit(msg)

				// send agree message
				msgJson, err := json.Marshal(msgReturn)
				if err == nil {
					s.SendMsg(message.ServerMsg{
						SType:      message.NODEMGMT,
						SendServer: s.ServerID.ID.Name,
						ReciServer: msgReturn.ReciNode,
						Payload:    msgJson,
					})
				}
				s.UpdateNodeInfo(-1)
			} else if msg.NMType == mgmt.NM_AGREE {
				// the exit node handle the AGREE message
				msgReturn := s.NodeManager.HandleAgree(msg)

				// send the RESTART message to the rest nodes
				if msgReturn != nil {
					msgJson, err := json.Marshal(msgReturn)
					if err == nil {
						go s.SendMsg(message.ServerMsg{
							SType:      message.NODEMGMT,
							SendServer: s.ServerID.ID.Name,
							ReciServer: msgReturn.ReciNode,
							Payload:    msgJson,
						})
					}
				}
			} else if msg.NMType == mgmt.NM_RESTART {

				// recieve the RESTART message, restart the orderer after waiting the signers update
				// wait new signer or the new threshold signer
				for !s.Orderer.IsReady() {
					time.Sleep(10 * time.Millisecond)
				}

				// restart the orderer
				s.RestartOrderer()

				// if the exit node is the leader of current view, need to refresh the leader after updating the node number
				if msg.SendNode == s.Orderer.GetLeaderName() {

					// refresh the leader
					s.Orderer.RefreshLeader()

					// the leader after refreshing init self for the new proposal
					if s.ServerID.ID.Name == s.Orderer.GetLeaderName() {
						s.Orderer.InitLeader()
					}
				}
			}
		}
	}
}
