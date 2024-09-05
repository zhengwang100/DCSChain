/*
PBFT in XBC

References
----------
Papers: << Practical Byzantine Fault Tolerance >> OSDI '99
Links: https://pmg.csail.mit.edu/papers/osdi99.pdf
*/
package core

import (
	"bcrequest"
	"blockchain"
	common "common"
	"encoding/json"
	"fmt"
	"log"
	"merkle"
	"message"
	"mgmt"
	"os"
	ptypes "pbft/types"
	ssm2 "ssm2"
	"strconv"
	"sync"
	"time"
)

// PBFT: the PBFT consensus core
type PBFT struct {
	CurPhase    ptypes.StateType // the consensus at which stage
	View        common.View      // the view consist of view number/leader number/nodes number
	ConsId      int              // the unique identity in consensus of the node
	SequenceNum int              // unique identification of the growing transaction sequence number within the system

	CurProposal  ptypes.Proposal // current proposal of this view
	ProposalLock sync.Mutex

	CheckPoint     ptypes.CheckPoint      // the unit of checkpoint maintained by this node,
	ViewChangeMsgs ptypes.MsgsLog         // the collection of view-change messages this node recieved
	NewViewMsgs    map[int][]*ptypes.PMsg // the collection of new-view messages this node recieved
	ReplyMsgs      []*ptypes.PMsg         // the collection of reply messages this node sent
	MsgLog         []ptypes.MsgsLog       // the collection of MsgLog,and there will be one for each view

	BlkStore    blockchain.BlockStore  // the unit to generate and store blocks
	PTimer      ptypes.PTimer          // the timer responsible for liveness
	ForwardChan chan []byte            // the channel through which this node receives messages can be responsible for sending messages from the consensus layer to the data layer
	SendChan    chan message.ServerMsg // the channel listened by a node can send messages in the channel to the corresponding node on the network
	Logger      log.Logger             `json:"logger"` // the role of recording logs
	Signer      *ssm2.Signer           `json:"Signer"` // the role responsible for signatures
}

// NewPBFT: create an instance of a new consensus of PBFT
// params:
// - timerDuration:	timeout period of the timer
// - consId:		the unique id of this orderer
// - nodeNum:		node number in system
// - path:			the path of block storage
// - senChan:		a channel provided by an upper-layer node through which messages can be sent
// - siger:			signer for sm2 sign
// return:
// - a new core of PBFT
func NewPBFT(timerDuration int, consId int, nodeNum int, path string,
	sendChan chan message.ServerMsg, signer *ssm2.Signer) *PBFT {
	newPBFTConsensus := &PBFT{
		CurPhase: ptypes.NEW_VIEW,
		View: common.View{
			ViewNumber: 0,
			NodesNum:   nodeNum,
			Leader:     0,
		},
		ConsId:      consId,
		NewViewMsgs: make(map[int][]*ptypes.PMsg),
		MsgLog:      make([]ptypes.MsgsLog, ptypes.CHECKPOINTNUM),
		Logger:      *log.New(os.Stdout, "", 0),
		BlkStore: blockchain.BlockStore{
			Base:       64,
			Height:     0,
			Path:       path + "\\r_" + strconv.Itoa(consId),
			PreBlkHash: merkle.EmptyHash(),
			CurBlkHash: merkle.EmptyHash(),
		},
		PTimer: ptypes.PTimer{
			VCMsgSendFlag: false,
			Timer:         common.NewTimer(time.Duration(timerDuration) * time.Millisecond),
		},
		CheckPoint: ptypes.CheckPoint{
			Seq:          0,
			CPMsgsBuffer: make(map[int][]*ptypes.PMsg),
			CPMsgs: []*ptypes.PMsg{{
				ViewNumber: 0,
				SeqNum:     0,
				SendNode:   "r_" + strconv.Itoa(consId),
			}},
		},
		SendChan: sendChan,
		Signer:   signer,
	}
	// set log format
	newPBFTConsensus.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	return newPBFTConsensus
}

// HandlePMsg: the node handle the message to consensus core and send its return message
// params:
// - msgJson: json of PBFT message
func (p *PBFT) HandlePMsg(payload []byte) {
	var msg ptypes.PMsg

	// convert json to message
	json.Unmarshal(payload, &msg)
	// fmt.Println(p.GetNodeName(),
	// 	p.View.ViewNumber,
	// 	msg.SendNode,
	// 	msg.ViewNumber,
	// 	msg.MType,
	// 	len(p.MsgLog[msg.ViewNumber%ptypes.CHECKPOINTNUM].PrepareMsgs),
	// 	len(p.MsgLog[msg.ViewNumber%ptypes.CHECKPOINTNUM].CommitMsgs))

	if msg.ViewNumber < p.View.ViewNumber && msg.MType != ptypes.CHECKPOINT {
		return
	}

	// submit the pbft message to pbft and get its return messages
	msgReturn := p.RoutePMsg(&msg)
	// fmt.Println(p.GetNodeName(), p.View.ViewNumber, msg.ViewNumber, msg.MType, msg.SendNode)

	// send its return messages and execute
	if msgReturn != nil {
		switch msgReturn.MType {

		// case reply message, the leader of the next node will prepare for next round
		case ptypes.REPLY:
			if p.Execute() {
				go p.SendSerMsg(&ptypes.PMsg{
					ViewNumber: p.View.ViewNumber - 1,
					// HsNode:     bhs.HsNode,
					Proposal: ptypes.Proposal{},
					SendNode: p.GetNodeName(),
					ReciNode: "Client",
				})

				// p.Logger.Println("[EXECUTE]:", p.GetNodeName(), "View:", p.View.ViewNumber-1)

				if p.IsLeader() {

					if p.CurPhase != ptypes.CHECKPOINT {
						p.CurPhase = ptypes.WAITING
					}

					go func() {
						if p.ProposalLock.TryLock() {
							msg := p.Preprepare()
							if msg != nil {
								msg.SendNode = p.GetNodeName()
								p.SendSerMsg(msg)
							}
							p.ProposalLock.Unlock()
						}
					}()
				}

				// if this reply's sequence is evenly divided by the const number CHECKPOINTNUM preset
				// and checkpoint message is not empty, the node will implement garbage collection mechanisms and update checkpoint
				if (msgReturn.SeqNum+1)%ptypes.CHECKPOINTNUM == 0 && len(p.CheckPoint.CPMsgsBuffer[msgReturn.SeqNum]) != 0 {
					p.SendSerMsg(p.CheckPoint.CPMsgsBuffer[msgReturn.SeqNum][0])
				}
				return
			}
		// case vc_reply message, which indicated the redo round after the view change
		case ptypes.VC_REPLY:

			// reply the redo request after viewchange
			for _, m := range msgReturn.OSet {
				for _, prePrepareMsg := range p.ViewChangeMsgs.NewViewMsgs[0].OSet {
					if m.SeqNum == prePrepareMsg.SeqNum {
						p.Execute()
						break
					}
				}
			}

			// view change finished successfully and reset the view change message log
			p.ReSetViewchangeMsgs()

			// the leader will start new view and prepare for new request
			if p.IsLeader() {
				p.CurPhase = ptypes.WAITING
				go func() {
					if p.ProposalLock.TryLock() {
						msg := p.Preprepare()
						if msg != nil {
							msg.SendNode = p.GetNodeName()
							p.SendSerMsg(msg)
						}
						p.ProposalLock.Unlock()
					}
				}()
			}
		default:
			// add node name and record it
			msgReturn.SendNode = p.GetNodeName()
			// n.PBFTConsensus.MsgLog[n.PBFTConsensus.ViewNumber%ptypes.CHECKPOINTNUM].SelfMsgs = append(n.PBFTConsensus.MsgLog[n.PBFTConsensus.ViewNumber%ptypes.CHECKPOINTNUM].SelfMsgs, &msg)
			p.SendSerMsg(msgReturn)
		}
	}
}

// RoutePMsg: the node in PBFT choose conresponding func to handle the message by message type
// params:
// - recieved message
// return:
// - message waiting to be sent
func (p *PBFT) RoutePMsg(msg *ptypes.PMsg) *ptypes.PMsg {
	switch msg.MType {
	case 0:
		return p.HandleNewViewMsg(msg)
	case 1:
		return p.HandlePrePrepareMsg(msg)
	case 2:
		return p.HandlePrepareMsg(msg)
	case 3:
		return p.HandleCommitMsg(msg)
	case 4:
		// fmt.Println("This is reply type message.", msg.MType, msg)
		return nil
	case 5:
		return p.HandleViewChangeMsg(msg)
	case 6:
		return p.HandleCheckPoint(msg)
	case 7:
		return p.HandleVCPrepareMsg(msg)
	case 8:
		return p.HandleVCCommitMsg(msg)
	default:
		fmt.Println("This is unknown type message: ", msg.MType, msg)
	}
	return nil
}

// HandleReq: the PBFT handle the request
// params:
// - height: 	block height
// - preHash: 	hash of previous block
// - req: 		recieved requests
func (p *PBFT) HandleReq(height int, preHash []byte, curHash []byte, req []bcrequest.BCRequest) {

	// generate a new proposal
	p.CurProposal = ptypes.Proposal{
		Height:     height,
		PreBlkHash: preHash,
		CurBlkHash: curHash,
		Command:    make([][]byte, 0),
	}
	for i := 0; i < len(req); i++ {
		p.CurProposal.Command = append(p.CurProposal.Command, req[i].Cmd)
	}

	// p.CurProposal.RootHash = merkle.HashFromByteSlicesIterative(p.CurProposal.Command)
	p.CurPhase = ptypes.NEW_VIEW
	p.ProposalLock.Lock()
	if p.CurPhase != ptypes.NEW_VIEW {
		return
	}
	// process the prepare phase of leader
	msgReturn := p.Preprepare()
	if msgReturn == nil {
		p.ProposalLock.Unlock()
		return
	}
	p.SendSerMsg(msgReturn)
	p.ProposalLock.Unlock()

}

// AddSyncInfo: add local information to message for sync, include view number/current block/hotstuff node/prepareQC/leader
// params:
// - msg: message which sync information needs to be added
func (p *PBFT) AddSyncInfo(msg *mgmt.NodeMgmtMsg) {
	msg.ViewNumber = p.View.ViewNumber
	msg.Leader = p.View.Leader
	msg.Block = append(msg.Block, p.BlkStore.CurProposalBlk)
	msg.HsNodes = append(msg.HsNodes, common.HsNode{
		CurHash:    []byte{byte(p.SequenceNum)},
		ParentHash: p.Signer.Pk,
	})
}

// SyncInfo: PBFT sync information from the selected sync-message
// params:
// - msg: the selected sync-message with sync information
// - leader: the leader of this view
func (p *PBFT) SyncInfo(msg *mgmt.NodeMgmtMsg, leader int) {
	p.Signer.ID = p.GetNodeName()
	p.Signer.Pks[p.Signer.ID] = p.Signer.Pk
	p.BlkStore.CurProposalBlk = msg.Block[0]
	p.BlkStore.Height = p.BlkStore.CurProposalBlk.BlkData.Height
	p.SequenceNum = int(msg.HsNodes[0].CurHash[0])

	// store the local block recieved
	p.BlkStore.StoreBlock(p.BlkStore.CurProposalBlk)
	// go to a new round and update

	// p.NewRound()
	p.View.UpdateView(msg.ViewNumber, leader)
}

// UpdateNodesNum: update the node num
// params:
// - nodeNum: the node number need to update
func (p *PBFT) UpdateNodesNum(nodeNum int) {
	p.View.NodesNum = nodeNum
}
