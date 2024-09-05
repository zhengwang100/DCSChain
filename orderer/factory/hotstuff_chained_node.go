package ofactory

import (
	"encoding/json"
	"fmt"
	hstypes "hotstuff/types"
	"local"
	"merkle"
)

// HandleMsg: the node handle the message to chained hotstuff core and send its return message
func (n *Node) CHandleMsg(ch chan []byte) {
	for {

		// constantly get messages from the channel
		if msgJson, ok := <-ch; ok {
			var msg hstypes.CMsg

			// convert json to message
			json.Unmarshal(msgJson, &msg)

			if n.ChainedHotstuff.ViewChangeSendFlag && msg.MType == hstypes.NEW_VIEW && msg.SendNode == n.NodeID.ID.Name {
				go n.SendCMsg(&msg)
				n.ChainedHotstuff.ViewChangeSendFlag = false
			}

			// submit the chained message to chained hotstuff and get its return messages
			msgReturnSlice := n.ChainedHotstuff.RouteCMsg(&msg)

			// send its return messages and execute
			if len(msgReturnSlice) != 0 {
				for _, msgReturn := range msgReturnSlice {
					if msgReturn == nil {
						continue
					}

					// add node name and record it
					msgReturn.SendNode = n.NodeID.ID.Name
					if msgReturn.MType == hstypes.GENERIC {
						n.ChainedHotstuff.CurRoundMsg = msgReturn
					}
					go n.SendCMsg(msgReturn)

					// execute cmds and store the proposal into local blockchain
					if msgReturn.MType == hstypes.NEW_VIEW && n.ChainedHotstuff.ExecuteState && len(n.ChainedHotstuff.Blocks[3].BlkHdr.Validation) != 0 {

						// if node successfully execute it, store it to blockchain
						if n.Execute() {
							// fmt.Println(time.Now())
							n.ChainedHotstuff.Logger.Println("[EXECUTE]", n.NodeID.ID.Name+" Success!", n.ChainedHotstuff.View.ViewNumber)
							// update the chained hotstuff execute state
							n.ChainedHotstuff.ExecuteState = false
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

// CHandleReq: the node recieve request and submit or transmit it
// BSesides, if existing not-execute proposal, it will submit an empty proposal for liveness
func (n *Node) CHandleReq() {
	for {
		// ensure the leader is waiting for a new proposal
		if n.ChainedHotstuff.CurPhase == hstypes.WAITING {
			// fmt.Println(n.ChainedHotstuff.ExistNotExecuteBlock(), n.NodeID.ID.Name)
			if len(n.Requests) != 0 && n.ChainedHotstuff.IsLeader() {
				n.ChainedHotstuff.CurProposal = hstypes.Proposal{
					Height:     n.BlkStore.Height,
					PreBlkHash: n.BlkStore.PreBlkHash,
					RootHash:   merkle.HashFromByteSlices(n.Requests),
					Commands:   n.Requests,
				}

				n.Requests = make([][]byte, 0)

				msgReturn := n.ChainedHotstuff.GenProposal()
				if msgReturn != nil {
					n.ChainedHotstuff.CurRoundMsg = msgReturn
					n.SendCMsg(msgReturn)
				}
			} else if n.ChainedHotstuff.ExistNotExecuteBlock() {
				// if exist not execute proposal, generate a dummy node

				n.ChainedHotstuff.CurProposal = hstypes.Proposal{}
				// n.Requests = make([][]byte, 0)

				// if exist not-execute proposal, change consensus state and continue new round
				if n.ChainedHotstuff.ExistNotExecuteBlock() {
					n.ChainedHotstuff.CurPhase = hstypes.NEW_VIEW
				}
				// time.Sleep(time.Second * 2)
			}
		}
	}
}

// SendCMsg: convert chained message to json and send it
// params:sendType(0: broadcast; 1: unicast; 2: gossip)
func (n *Node) SendCMsg(msg *hstypes.CMsg) {
	msgJson, err := json.Marshal(msg)
	if err == nil {
		if msg.ReciNode == "Broadcast" {
			local.Broadcast(n.NodeManager.NodesChannel, msgJson, n.NodeID.ID.Name)
			// n.ChainedHotstuff.Logger.Println("[Broadcast]", msg.MType, n.NodeID.ID.Name+" ->", n.GetNodeNames())
		} else {
			local.Unicast(n.NodeManager.NodesChannel, msgJson, msg.ReciNode, n.NodeID.ID.Name)
			// n.ChainedHotstuff.Logger.Println("[Unicast]", msg.MType, n.NodeID.ID.Name+" ->", msg.ReciNode)
		}
	}
}
