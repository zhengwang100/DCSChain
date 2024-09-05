package ofactory

import (
	"encoding/json"
	"fmt"
	hs2types "hotstuff2/types"
	"local"
	"merkle"
)

// H2HandleMsg: the node handle the message to hotstuff-2 core and send its return message
func (n *Node) H2HandleMsg(ch chan []byte) {
	for {

		// constantly get messages from the channel
		if msgJson, ok := <-ch; ok {
			var msg hs2types.H2Msg

			// convert json to message
			json.Unmarshal(msgJson, &msg)
			// fmt.Println(msg.MType, msg.SendNode, msg.ReciNode)

			if msg.MType == hs2types.WISH && n.Hotstuff2.PM.WishSendFlag && msg.SendNode == n.NodeID.ID.Name {
				n.Hotstuff2.PM.WishSendFlag = false
				n.SendH2Msg(&msg)
				continue
			}

			// submit the chained message to hotstuff-2 and get its return messages
			msgReturn := n.Hotstuff2.RouteH2Msg(&msg)

			// send its return messages and execute
			if msgReturn != nil {

				// add node name and record it
				msgReturn.SendNode = n.NodeID.ID.Name
				n.Hotstuff2.CurRoundMsgs = append(n.Hotstuff2.CurRoundMsgs, msgReturn)
				n.SendH2Msg(msgReturn)
			}
		} else {
			fmt.Println("通道已关闭，没有数据了")
			break
		}
	}
}

// H2HadnleReq: the node recieve request and submit or transmit it
// BSesides, if existing not-execute proposal, it will submit an empty proposal for liveness
func (n *Node) H2HandleReq() {
	for {
		// ensure the leader is waiting for a new proposal
		if len(n.Requests) != 0 && n.Hotstuff2.IsLeader() {
			curProposal := hs2types.Proposal{
				Height:     n.BlkStore.Height,
				PreBlkHash: n.BlkStore.PreBlkHash,
				RootHash:   merkle.HashFromByteSlices(n.Requests),
				Command:    n.Requests,
			}
			n.Hotstuff2.CurProposal = curProposal
			n.Requests = make([][]byte, 0)
			msgReturn := n.Hotstuff2.GenProposal(&hs2types.H2Msg{})
			if msgReturn != nil {
				n.SendH2Msg(msgReturn)
			}
		}
	}
}

// SendH2Msg: convert hotstuff-2 message to json and send it
// params:sendType(0: broadcast; 1: unicast; 2: gossip)
func (n *Node) SendH2Msg(msg *hs2types.H2Msg) {
	msgJson, err := json.Marshal(msg)
	if err == nil {
		if msg.ReciNode == "Broadcast" {
			local.Broadcast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.Hotstuff2.Logger.Println("[Broadcast]", msg.MType, n.NodeID.ID.Name+" ->", n.GetNodeNames())
		} else if msg.ReciNode == "Gossip" {
			local.Gossip(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.Hotstuff2.Logger.Println("[Gossip]", msg.MType, n.NodeID.ID.Name+" ->", n.GetNodeNames())
		} else if msg.ReciNode == "Forward" {
			local.Unicast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name, n.NodeID.ID.Name)
			// n.Hotstuff2.Logger.Println("[Forward]", msg.MType, n.NodeID.ID.Name)
		} else {
			local.Unicast(n.NodeManager.NodesChannel, msgJson, msg.ReciNode, n.NodeID.ID.Name)
			// n.Hotstuff2.Logger.Println("[Unicast]", msg.MType, n.NodeID.ID.Name+" ->", msg.ReciNode)
		}
	}
}
