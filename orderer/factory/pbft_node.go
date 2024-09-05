package ofactory

import (
	"encoding/json"
	"fmt"
	"local"
	ptypes "pbft/types"
)

// PHandleMsg: the PBFT node process the message
func (n *Node) PHandleMsg(ch chan []byte) {
	for {

		// constantly get messages from the channel
		if msgJson, ok := <-ch; ok {
			var msg ptypes.PMsg

			// convert json to message
			json.Unmarshal(msgJson, &msg)

			if n.PBFTConsensus.PTimer.VCMsgSendFlag && msg.SendNode == n.NodeID.ID.Name && msg.MType == ptypes.VIEW_CHANGE {
				// fmt.Println(n.NodeID.ID.Name, len(n.PBFTConsensus.NewViewMsgs))
				n.PBFTConsensus.PTimer.VCMsgSendFlag = false
				// fmt.Println("Pmsgprocess", msg, msgJson)
				n.SendPMsg(&msg)
				continue
			}

			// submit the pbft message to pbft and get its return messages
			msgReturn := n.PBFTConsensus.RoutePMsg(&msg)

			// send its return messages and execute
			if msgReturn != nil {
				switch msgReturn.MType {

				// case reply message, the leader of the next node will prepare for next round
				case ptypes.REPLY:
					if n.Execute() {
						n.PBFTConsensus.Logger.Println("[EXECUTE]:", n.NodeID.ID.Name, "View:", n.PBFTConsensus.View.ViewNumber-1)
						if n.PBFTConsensus.IsLeader() {
							go func() {
								msg := n.PBFTConsensus.Preprepare()
								if msg != nil {
									msg.SendNode = n.NodeID.ID.Name
									n.SendPMsg(msg)
								}
							}()
						}

						// if this reply's sequence is evenly divided by the const number CHECKPOINTNUM preset
						// and checkpoint message is not empty, the node will implement garbage collection mechanisms and update checkpoint
						if (msgReturn.SeqNum+1)%ptypes.CHECKPOINTNUM == 0 && len(n.PBFTConsensus.CheckPoint.CPMsgsBuffer[msgReturn.SeqNum]) != 0 {
							n.SendPMsg(n.PBFTConsensus.CheckPoint.CPMsgsBuffer[msgReturn.SeqNum][0])
						}
						continue
					}
				// case vc_reply message, which indicated the redo round after the view change
				case ptypes.VC_REPLY:

					// reply the redo request after viewchange
					for _, m := range msgReturn.OSet {
						for _, prePrepareMsg := range n.PBFTConsensus.ViewChangeMsgs.NewViewMsgs[0].OSet {
							if m.SeqNum == prePrepareMsg.SeqNum {
								n.Execute()
								break
							}
						}
					}

					// view change finished successfully and reset the view change message log
					n.PBFTConsensus.ReSetViewchangeMsgs()

					// the leader will start new view and prepare for new request
					if n.PBFTConsensus.IsLeader() {
						go func() {
							msg := n.PBFTConsensus.Preprepare()
							if msg != nil {
								msg.SendNode = n.NodeID.ID.Name
								n.SendPMsg(msg)
							}
						}()
					}
				default:
					// add node name and record it
					msgReturn.SendNode = n.NodeID.ID.Name
					// n.PBFTConsensus.MsgLog[n.PBFTConsensus.ViewNumber%ptypes.CHECKPOINTNUM].SelfMsgs = append(n.PBFTConsensus.MsgLog[n.PBFTConsensus.ViewNumber%ptypes.CHECKPOINTNUM].SelfMsgs, &msg)
					n.SendPMsg(msgReturn)
				}
			}
		} else {
			fmt.Println("通道已关闭，没有数据了")
			break
		}
	}
}

// H2HadnleReq: the node recieve request and submit or transmit it
// BSesides, if existing not-execute proposal, it will submit an empty proposal for liveness
func (n *Node) PHandleReq() {
	for {
		// ensure the leader is waiting for a new proposal
		// fmt.Println(len(n.Requests) != 0, n.NodeID.ID.Name)
		if len(n.Requests) != 0 && n.PBFTConsensus.IsLeader() {

			curProposal := ptypes.Proposal{
				Height:     n.BlkStore.Height,
				PreBlkHash: n.BlkStore.PreBlkHash,
				CurBlkHash: n.BlkStore.CurBlkHash,
				Command:    n.Requests,
			}
			n.PBFTConsensus.CurProposal = curProposal
			n.Requests = make([][]byte, 0)
			n.PBFTConsensus.CurPhase = ptypes.NEW_VIEW
			msgReturn := n.PBFTConsensus.Preprepare()
			if msgReturn != nil {
				n.SendPMsg(msgReturn)
			}
		}
	}
}

// SendPMsg: local send PBFT message
func (n *Node) SendPMsg(msg *ptypes.PMsg) {
	msgJson, err := json.Marshal(msg)
	if err == nil {
		if msg.ReciNode == "Broadcast" {
			local.Broadcast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.PBFTConsensus.Logger.Println("[Broadcast]", msg.MType, n.NodeID.ID.Name+" ->", n.GetNodeNames())
		} else if msg.ReciNode == "Gossip" {
			local.Gossip(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.PBFTConsensus.Logger.Println("[Gossip]", msg.MType, n.NodeID.ID.Name+" ->", n.GetNodeNames())
		} else if msg.ReciNode == "Forward" {
			local.Unicast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name, n.NodeID.ID.Name)
			// n.PBFTConsensus.Logger.Println("[Forward]", msg.MType, n.NodeID.ID.Name)
		} else {
			local.Unicast(n.NodeManager.NodesChannel, msgJson, msg.ReciNode, n.NodeID.ID.Name)
			// n.PBFTConsensus.Logger.Println("[Unicast]", msg.MType, n.NodeID.ID.Name+" ->", msg.ReciNode)
		}
	}
}
