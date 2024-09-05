package ofactory

import (
	"encoding/json"
	"fmt"
	hstypes "hotstuff/types"
	"local"
	"merkle"
	"mgmt"
	"time"
)

// HandleMsg: the node handle the message to different core
func (n *Node) HandleMsg(ch chan []byte) {
	for {
		// constantly get messages from the channel
		if msgJson, ok := <-ch; ok {
			var msg hstypes.Msg
			// convert json to message
			err := json.Unmarshal(msgJson, &msg)
			if err == nil {
				if n.HandleState {
					// if n.HandleState {
					// when handle state is false, means the orderer stop the node
					n.HandleConsMsg(msg)
				}
			} else {
				var msg mgmt.NodeMgmtMsg

				// convert json to message
				err := json.Unmarshal(msgJson, &msg)

				if err != nil {
					fmt.Println("ERROR", err)
				}
				if msg.Type == mgmt.JOIN {
					if msg.NMType == mgmt.NM_SYNC {
						msgSync, msgReturn := n.NodeManager.HandleSync(&msg)
						if msgSync != -1 && msgReturn != nil && n.NodeManager.State == mgmt.NM_SYNC {
							n.UpdateNodeInfo(msgSync)
							n.SendNMMsg(msgReturn)
						}
					} else if msg.NMType == mgmt.NM_RESTART {
						for n.BasicHotstuff.View.NodesNum != n.BasicHotstuff.ThresholdSigner.SignNum {
							time.Sleep(10 * time.Millisecond)
						}
						n.RestartNode()
					} else {

						// The original node stop and send sync message
						n.Stop()
						msgReturn := n.NodeManager.HandleJoin(&msg)
						msgReturn.ViewNumber = n.BasicHotstuff.View.ViewNumber
						msgReturn.Block[0] = n.BasicHotstuff.BlkStore.CurProposalBlk
						msgReturn.HsNodes[0] = n.BasicHotstuff.HsNode
						msgReturn.Justify = n.BasicHotstuff.PrepareQC
						msgReturn.Leader = n.BasicHotstuff.View.Leader
						n.SendNMMsg(msgReturn)
						n.UpdateNodeInfo(-1)
					}
				} else if msg.Type == mgmt.EXIT {
					if msg.NMType == mgmt.NM_APPLY {
						// The original node stop and send sync message
						n.Stop()
						msgReturn := n.NodeManager.HandleExit(&msg)
						n.SendNMMsg(msgReturn)
						n.UpdateNodeInfo(-1)
					} else if msg.NMType == mgmt.NM_AGREE {
						msgReturn := n.NodeManager.HandleAgree(&msg)
						if msgReturn != nil {
							n.SendNMMsg(msgReturn)
						}
					} else if msg.NMType == mgmt.NM_RESTART {
						for n.BasicHotstuff.View.NodesNum != n.BasicHotstuff.ThresholdSigner.SignNum {
							time.Sleep(10 * time.Millisecond)
						}
						n.RestartNode()
						if msg.SendNode == n.BasicHotstuff.GetLeaderName() {
							// n.BasicHotstuff.IgnoreCheckQC = true
							n.BasicHotstuff.View.RefreshLeader()
						}
						if n.NodeID.ID.Name == n.BasicHotstuff.GetLeaderName() {
							n.BasicHotstuff.InitLeader()
						}
					}
				}

			}

		} else {
			fmt.Println("通道已关闭，没有数据了")
			break
		}
	}
}

// HandleConsMsg: the node handle the message to consensus core and send its return message
func (n *Node) HandleConsMsg(msg hstypes.Msg) {

	if n.BasicHotstuff.ViewChangeSendFlag && msg.MType == hstypes.NEW_VIEW && msg.SendNode == n.NodeID.ID.Name {
		n.SendBMsg(&msg)
		n.BasicHotstuff.ViewChangeSendFlag = false
		return
	}

	// submit the message to basic hotstuff and get its return messages
	msgReturn := n.BasicHotstuff.RouteBMsg(&msg, nil)
	if msgReturn != nil {
		msgReturn.SendNode = n.NodeID.ID.Name
		n.BasicHotstuff.CurRoundMsg = append(n.BasicHotstuff.CurRoundMsg, msgReturn)
		n.SendBMsg(msgReturn)

		// execute cmds and store the proposal into local blockchain
		if msgReturn.MType == hstypes.NEW_VIEW && !n.BasicHotstuff.CurProposal.IsEmpty() {

			// if node successfully execute it, store it to blockchain
			if n.Execute() {

				n.BasicHotstuff.Logger.Println("[EXECUTE]", n.NodeID.ID.Name+" Success!")

				// n.NodeManager.MsgLog = n.BasicHotstuff.LastRoundMsg
				n.BasicHotstuff.UpdateBasicHotstuff()
			}
		}
	}
}

// HandleReq: the node recieve request and submit or transmit it
func (n *Node) HandleReq() {
	for n.ReqState {
		// in the basic hotstuff, only the leader put forward a proposal
		if len(n.Requests) != 0 && n.NodeID.ID.Name == n.BasicHotstuff.GetLeaderName() {
			// if len(n.Requests) != 0 && n.BasicHotstuff.IsLeader() {
			n.BasicHotstuff.CurProposal = hstypes.Proposal{
				Height:     n.BlkStore.Height,
				PreBlkHash: n.BlkStore.PreBlkHash,
				RootHash:   merkle.HashFromByteSlicesIterative(n.Requests),
				Commands:   n.Requests,
			}
			n.Requests = make([][]byte, 0)
			// time.Sleep(time.Second * 2)

			if n.BasicHotstuff.CurPhase == hstypes.WAITING {
				msgReturn := n.BasicHotstuff.GenProposal()

				if msgReturn != nil {
					msgReturn.SendNode = n.NodeID.ID.Name
					n.BasicHotstuff.CurRoundMsg = append(n.BasicHotstuff.CurRoundMsg, msgReturn)
					n.SendBMsg(msgReturn)
				}
			}
		}
	}
}

// SendMsg: convert message to json and send it
// params:sendType(0: broadcast; 1: unicast; 2: gossip)
func (n *Node) SendBMsg(msg *hstypes.Msg) {
	msgJson, err := json.Marshal(msg)
	if err == nil {
		switch msg.ReciNode {
		case "Broadcast":
			local.Broadcast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.BasicHotstuff.Logger.Println("[Broadcast]", n.NodeID.ID.Name+" ->", n.NodeManager.GetNodeNames())
		case "Gossip":
			local.Gossip(n.NodeManager.NodesChannel, msgJson, msg.SendNode)
			// n.BasicHotstuff.Logger.Println("[Gossip]", n.NodeID.ID.Name+" ->", n.NodeManager.GetOtherNodeNames())
		case n.NodeManager.NewNode.Name:
			local.Fixedcast(n.NodeManager.NewNode.Chan, msgJson)
			// n.BasicHotstuff.Logger.Println("[Fixedcast]", n.NodeID.ID.Name+" ->", msg.ReciNode)
		default:
			local.Unicast(n.NodeManager.NodesChannel, msgJson, msg.ReciNode, n.NodeID.ID.Name)
			// n.BasicHotstuff.Logger.Println("[Unicast]", n.NodeID.ID.Name+" ->", msg.ReciNode)
		}
	}
}

// SendMsg: convert message to json and send it
// params:sendType(0: broadcast; 1: unicast; 2: gossip)
func (n *Node) SendNMMsg(msg *mgmt.NodeMgmtMsg) {
	msgJson, err := json.Marshal(msg)
	if err == nil {
		switch msg.ReciNode {
		case "Broadcast":
			local.Broadcast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.BasicHotstuff.Logger.Println("[Broadcast]", n.NodeID.ID.Name+" ->", n.NodeManager.GetNodeNames())
		case "Gossip":
			local.Gossip(n.NodeManager.NodesChannel, msgJson, msg.SendNode)
			// n.BasicHotstuff.Logger.Println("[Gossip]", n.NodeID.ID.Name+" ->", n.NodeManager.GetOtherNodeNames())
		case n.NodeManager.NewNode.Name:
			local.Fixedcast(n.NodeManager.NewNode.Chan, msgJson)
			// n.BasicHotstuff.Logger.Println("[Fixedcast]", n.NodeID.ID.Name+" ->", msg.ReciNode)
		default:
			local.Unicast(n.NodeManager.NodesChannel, msgJson, msg.ReciNode, n.NodeID.ID.Name)
			// n.BasicHotstuff.Logger.Println("[Unicast]", n.NodeID.ID.Name+" ->", msg.ReciNode)
		}
	}
}

// GetLeaderNum: the node get the leader number of this view
func (n *Node) GetLeaderNum() int {
	return int(n.BasicHotstuff.View.ViewNumber) % len(n.NodeManager.NodesTable)
}

// RestartNode: restart the node
func (n *Node) RestartNode() {
	// clear the messages in current round
	n.BasicHotstuff.ClearCurrentRound()
	n.HandleState = true
	n.ReqState = true

	// start a process to accept the request
	go n.HandleReq()

	if n.BasicHotstuff.GetLeaderName() == n.NodeID.ID.Name {
		if !n.BasicHotstuff.CurProposal.IsEmpty() {
			msgReturn := n.BasicHotstuff.GenProposal()

			// fmt.Println(n.NodeID.ID.Name, msgReturn)
			if msgReturn != nil {
				msgReturn.SendNode = n.NodeID.ID.Name
				n.BasicHotstuff.CurRoundMsg = append(n.BasicHotstuff.CurRoundMsg, msgReturn)
				n.SendBMsg(msgReturn)
			}
		} else {
			n.BasicHotstuff.CurPhase = hstypes.WAITING
		}
	}
}

// Stop: stop the leader, update the orderer HandleState and ReqState to false, stop timer
func (n *Node) Stop() {
	n.HandleState = false
	n.ReqState = false
	n.BasicHotstuff.ViewTimer.Stop()
}
