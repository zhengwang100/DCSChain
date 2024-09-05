/*
HotStuff-2 in XBC

References
----------
Papers: << HotStuff-2: Optimal Two-Phase Responsive BFT >>
Links: https://eprint.iacr.org/2023/397.pdf
*/
package core

import (
	"bcrequest"
	"blockchain"
	common "common"
	"encoding/json"
	"fmt"
	"hotstuff2/pacemaker"
	hs2types "hotstuff2/types"
	"log"
	"merkle"
	"message"
	"mgmt"
	"os"
	"strconv"
	"sync"
	"time"
	"tss"
)

// Hotstuff2: the hotstuff-2 consensus core
type Hotstuff2 struct {
	CurPhase hs2types.StateType // the consensus at which stage
	View     common.View        //
	ConsId   int                // the unique identity in consensus of the node

	CurHs2Node   common.HsNode     // current hotstuff-2 node of this view, which is consist of hash of current block and its parent block
	CurProposal  hs2types.Proposal // current proposal of this view
	ProposalLock sync.Mutex
	ProposalQC   hs2types.QuromCert // highest locked single certification, the name and meaning is the same as hotstuff
	PrepareQC    hs2types.QuromCert // highest locked double certification, the name and meaning is the same as hotstuff

	LockBlk     []*blockchain.Block // local locked block which is consist of all blocks that have been voted(refer to Vote2) but have not yet been committed
	LockHs2Node []common.HsNode     // local locked Node which is consist of all Nodes that have been voted(refer to Vote2) but have not yet been committed

	IgnoreCheckQC bool // the flag ignore the effectiveness of QC

	CurRoundMsgs []*hs2types.H2Msg // the collection of messages sent by this node in the current view
	LastRoundMsg []*hs2types.H2Msg // the collection of messages sent by this node in the last view
	NewViewMsgs  []*hs2types.H2Msg // the collection of new-view messages this node recieved
	Vote1        []*hs2types.H2Msg // the collection of vote1 messages this node recieved
	Vote2        []*hs2types.H2Msg // the collection of vote2 messages this node recieved

	BlkStore        blockchain.BlockStore  // generate and store blocks
	PM              pacemaker.Pacemaker    // the pacemaker in the same paper controls the activity of consensus
	ForwardChan     chan []byte            // the channel through which this node receives messages can be responsible for sending messages from the consensus layer to the data layer
	SendChan        chan message.ServerMsg // the channel listened by a node can send messages in the channel to the corresponding node on the network
	Logger          log.Logger             `json:"logger"` // the role of recording logs
	ThresholdSigner *tss.Signer            `json:"Signer"` // the role responsible for threshold signatures
}

// NewHotstuff2: create an instance of a new consensus of hotstuff-2
// params:
// - enterTD:	timeout period of the enter-timer
// - viewTD:	timeout period of the view-timer
// - consId:	the unique id of this orderer
// - nodeNum:	node number in system
// - path:		the path of block storage
// - senChan:	a channel provided by an upper-layer node through which messages can be sent
// - siger:		signer for threshold sign
// return:
// - a new core of hostuff-2
func NewHotstuff2(enterTD int, viewTD int, consId int, nodeNum int, path string, sendChan chan message.ServerMsg, signer *tss.Signer) *Hotstuff2 {
	newHotstuff2 := Hotstuff2{
		CurPhase: hs2types.NEW_VIEW,
		View: common.View{
			ViewNumber: 0,
			NodesNum:   nodeNum,
			Leader:     0,
		},
		ConsId: consId,

		Logger: *log.New(os.Stdout, "", 0),
		PM: pacemaker.Pacemaker{
			OptimisticFlag: false,
			EnterTimer:     *common.NewTimer(time.Duration(enterTD) * time.Millisecond),
			ViewTimer:      *common.NewTimer(time.Duration(viewTD) * time.Millisecond),
			WishMsgs:       make(map[int][]*hs2types.H2Msg),
		},
		BlkStore: blockchain.BlockStore{
			Base:   64,
			Height: 0,
			Path:   path + "\\r_" + strconv.Itoa(consId),
		},
		ThresholdSigner: signer,
		SendChan:        sendChan,
		IgnoreCheckQC:   false,
	}

	newHotstuff2.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	return &newHotstuff2
}

// HandleH2Msg: the node handle the message to consensus core and send its return message
// params:
// - msgJson: json of hostuff-2 message
func (hs2 *Hotstuff2) HandleH2Msg(msgJson []byte) {
	var msg hs2types.H2Msg

	// convert json to message
	json.Unmarshal(msgJson, &msg)
	// fmt.Println(msg.MType, msg.SendNode, msg.ReciNode)

	// submit the chained message to hotstuff-2 and get its return messages
	msgReturn := hs2.RouteH2Msg(&msg)

	// send its return messages and execute
	if msgReturn != nil {

		// add node name and record it
		msgReturn.SendNode = hs2.GetNodeName()
		hs2.CurRoundMsgs = append(hs2.CurRoundMsgs, msgReturn)
		if msgReturn.MType == hs2types.ENTER {
			go hs2.SendSerMsg(&hs2types.H2Msg{
				ViewNumber: hs2.View.ViewNumber - 1,
				// Hs2Node:    hs2.LockHs2Node[0],
				SendNode: hs2.GetNodeName(),
				ReciNode: "Client",
			})
		}

		hs2.SendSerMsg(msgReturn)
	}

}

// RouteH2MsgMsg: the node in hotstuff-2 choose conresponding func to handle the message by message type
// params:
// - msg: recieved message
// return:
// - message waiting to be sent
func (hs2 *Hotstuff2) RouteH2Msg(msg *hs2types.H2Msg) *hs2types.H2Msg {
	// fmt.Println("bshs HandleMsg", msg.MType)
	switch msg.MType {
	case 0:
		return hs2.HandleNewView(msg)
	case 1:
		return hs2.GenProposal(msg)
	case 2:
		return hs2.HandlePropose(msg) // vote1
	case 3:
		return hs2.HandleVote1(msg)
	case 4:
		return hs2.HandlePrepare(msg) // vote2
	case 5:
		return hs2.HandleVote2(msg)
	case 6:
		return hs2.HandleWish(msg)
	case 7:
		return hs2.HandleTC(msg)
	case 8:
		return hs2.HandleOptimisticEnter(msg)
	default:
		fmt.Println("The type of this message isn't included konw type")
		return nil
	}
}

// HandleReq: the hotstuff-2 handle the request
// params:
// - height: 	block height
// - preHash: 	hash of previous block
// - req: 		recieved requests
func (hs2 *Hotstuff2) HandleReq(height int, preHash []byte, req []bcrequest.BCRequest) {
	hs2.CurProposal = hs2types.Proposal{
		Height:     height,
		PreBlkHash: preHash,
		Command:    make([][]byte, 0),
	}
	for i := 0; i < len(req); i++ {
		hs2.CurProposal.Command = append(hs2.CurProposal.Command, req[i].Cmd)
	}

	hs2.CurProposal.RootHash = merkle.HashFromByteSlicesIterative(hs2.CurProposal.Command)
	// fmt.Println("HandleReq", hs2.CurPhase, hs2.View.ViewNumber, hs2.GetNodeName())

	// hs2.ProposalLock.Lock()
	// defer hs2.ProposalLock.Unlock()

	if hs2.CurPhase == hs2types.NEW_PROPOSE {
		msgReturn := hs2.GenProposal(&hs2types.H2Msg{
			MType:    hs2types.NEW_PROPOSE,
			SendNode: hs2.GetNodeName(),
		})
		if msgReturn == nil {
			return
		}
		hs2.CurPhase = hs2types.PROPOSE
		hs2.SendSerMsg(msgReturn)
	}
}

// RestartHotstuff2: restart the hotstuff-2 to handle the request and message
// return:
// - the prepare message
func (hs2 *Hotstuff2) RestartHotstuff2() *hs2types.H2Msg {
	if hs2.GetLeaderName() == hs2.GetNodeName() {
		if !hs2.CurProposal.IsEmpty() {
			msgReturn := hs2.GenProposal(&hs2types.H2Msg{
				MType: hs2types.NEW_PROPOSE,
			})

			if msgReturn != nil {
				hs2.CurRoundMsgs = append(hs2.CurRoundMsgs, msgReturn)
			}
			return msgReturn
		}
	}
	return nil

}

// IsLeader: check whether the node is leader
func (hs2 *Hotstuff2) IsLeader() bool {
	return hs2.GetLeaderNum() == hs2.ConsId
}

// CreateLeaf: create new hotstuff-2 node extend from parent
func (hs2 *Hotstuff2) CreateLeaf(parent []byte) common.HsNode {
	return common.HsNode{
		ParentHash: parent,
		CurHash:    hs2.BlkStore.CurBlkHash,
	}
}

// CheckQC: check message's QC is valid, include QC's type, view, height and signature
func (hs2 *Hotstuff2) CheckQC(qc *hs2types.QuromCert, code hs2types.StateType, viewNumber int, height int) bool {

	if !MatchingQC(qc, code, viewNumber) {
		fmt.Println("MatchingQC r_", hs2.ConsId, " in the ", hs2.CurPhase, "within view ", hs2.View.ViewNumber, "check qc.QType:", qc.QType, "code:", code, "vn:", viewNumber, "qc.vn:", qc.ViewNumber)
		return false
	}

	if qc.Height != height {
		fmt.Println("CheckQC height matchting Error", qc.QType, qc.Height, height)
		return false
	}

	if !hs2.ThresholdSigner.ThresholdSignVerify(qc.QC2SignMsgByte(), qc.Sign) {
		fmt.Println("CheckQC ThresholdSignVerify Error")
		return false
	}
	return true
}

// MatchingQC: check QC's type and view
// params: qurom certfication, qc's type, current view
// return: result
func MatchingQC(qc *hs2types.QuromCert, code hs2types.StateType, curView int) bool {
	// fmt.Println(qc.QType, code, qc.ViewNumber, curView)
	if qc.QType == code && qc.ViewNumber == curView {
		return true
	}
	return false
}

// CheckMsg: check message according message's type
func CheckMsg(msg *hs2types.H2Msg, code hs2types.StateType, viewNum int) bool {
	switch code {
	case hs2types.NEW_VIEW:
		return MatchingMsg(msg, code, viewNum-1)
	case hs2types.PROPOSE, hs2types.VOTE1, hs2types.PREPARE, hs2types.VOTE2:
		return MatchingMsg(msg, code, viewNum)
	default:
		return false
	}
}

// MatchingMsg: check whether message's type and view are matching
func MatchingMsg(msg *hs2types.H2Msg, code hs2types.StateType, curView int) bool {
	return msg.MType == code && msg.ViewNumber == curView
}

// CombineSig: combine message's part signature to a complete signature
// params: the silce of recieved messages with part signature
// ruturn: byte silce of the complete signature
func (hs2 *Hotstuff2) CombineSign(voteMsgs []*hs2types.H2Msg, msgSign hs2types.H2Msg) []byte {

	// declare a two-dimensional byte slices for collecting part signatures
	var partSigs [][]byte
	for _, m := range voteMsgs {
		partSigs = append(partSigs, m.ConsSign)
	}

	// combine the complete signature according to part signature and recovered messages
	sig, err := hs2.ThresholdSigner.CombineSig(msgSign.Message2Byte(), partSigs)
	if err == nil {
		return sig
	}
	fmt.Println("error hs2 combineSign", hs2.ConsId, err)
	return nil
}

// UpdateProposal: update local proposals with new proposal
func (hs2 *Hotstuff2) UpdateConsensus() {
	hs2.LockHs2Node = append(hs2.LockHs2Node, hs2.CurHs2Node)
	hs2.LockBlk = append(hs2.LockBlk, &blockchain.Block{
		BlkHdr:  hs2.BlkStore.CurProposalBlk.BlkHdr,
		BlkData: hs2.BlkStore.CurProposalBlk.BlkData,
	})
	hs2.BlkStore.GenEmptyBlock()
	hs2.CurProposal = hs2types.Proposal{}
	hs2.CurHs2Node = common.HsNode{}

	hs2.NewViewMsgs = make([]*hs2types.H2Msg, 0)
	hs2.Vote1 = make([]*hs2types.H2Msg, 0)
	hs2.Vote2 = make([]*hs2types.H2Msg, 0)
}

// GetLeaderNum: the node get the leader number of this view
func (hs2 *Hotstuff2) GetLeaderNum() int {
	return hs2.View.Leader
}

// WriteNewHs2Block: write new hotstuff-2 block to local
// params: the write block index range [start, end] in the locked block
func (hs2 *Hotstuff2) WriteNewH2Block(start int, end int) {
	// if the start or end is -1, don't need write block and return directly
	// fmt.Println(start, end)
	if start == -1 || end == -1 {
		return
	}

	// from start to end write blocks in order
	for i := start; i <= end; i++ {
		hs2.BlkStore.StoreBlock(*hs2.LockBlk[i])
	}
	hs2.UpdateAfterCommit(start, end)
	hs2.CurProposal.PreBlkHash = hs2.BlkStore.PreBlkHash
}

// UpdateAfterCommit; update locked Hs2Node and block by delete commited and stored blk after commit
// params: the index range [start,end] of the locked block to be deleted
func (hs2 *Hotstuff2) UpdateAfterCommit(start int, end int) {
	if start == 0 && end == len(hs2.LockHs2Node)-1 {
		hs2.LockHs2Node = make([]common.HsNode, 0)
		hs2.LockBlk = make([]*blockchain.Block, 0)
	} else if start == 0 && end == 0 {

	} else if start == 0 {
		hs2.LockHs2Node = hs2.LockHs2Node[end+1:]
		hs2.LockBlk = hs2.LockBlk[end+1:]
	} else if end == len(hs2.LockHs2Node)-1 {
		hs2.LockHs2Node = hs2.LockHs2Node[:start]
		hs2.LockBlk = hs2.LockBlk[:start]
	} else {
		hs2.LockHs2Node = append(hs2.LockHs2Node[:start], hs2.LockHs2Node[end+1:]...)
		hs2.LockBlk = append(hs2.LockBlk[:start], hs2.LockBlk[end+1:]...)
	}
}

// GetNodeName: get the node name by the ID
func (hs2 *Hotstuff2) GetNodeName() string {
	return "r_" + strconv.Itoa(hs2.ConsId)
}

// GetNodeName: get the next leader node name by the ID
func (hs2 *Hotstuff2) GetNextLeaderName() string {
	return hs2.View.NextLeaderName()
}

// GetLeaderName: get leader of current view name
func (hs2 *Hotstuff2) GetLeaderName() string {
	return hs2.View.LeaderName()
}

// SendSerMsg: send the message, in fact the chan provided by the outer layer is passed to the outer layer,
// and the outer layer sends the message
// params:
// - msg: message that need to be sent
func (hs2 *Hotstuff2) SendSerMsg(msg *hs2types.H2Msg) {
	msgJson, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	serMsg := message.ServerMsg{
		SType:      message.ORDER,
		SendServer: msg.SendNode,
		ReciServer: msg.ReciNode,
		Payload:    msgJson,
	}
	hs2.SendChan <- serMsg
}

// initLeader: the node which is leader in view init
func (hs2 *Hotstuff2) InitLeader() {
	hs2.CurPhase = hs2types.NEW_PROPOSE
}

// AddSyncInfo: add local information to message for sync, include view number/current block/hotstuff node/prepareQC/leader
// params:
// - msg: message which sync information needs to be added
func (hs2 *Hotstuff2) AddSyncInfo(msg *mgmt.NodeMgmtMsg) {
	msg.ViewNumber = hs2.View.ViewNumber
	// msg.Block = append(msg.Block, hs2.BlkStore.CurProposalBlk)
	count := len(hs2.LockBlk)
	for i := 0; i < count; i++ {
		msg.Block = append(msg.Block, *hs2.LockBlk[i])
	}
	msg.HsNodes = append(msg.HsNodes, hs2.CurHs2Node)
	msg.HsNodes = append(msg.HsNodes, hs2.LockHs2Node...)
	msg.H2Justify = hs2.PrepareQC
	msg.Leader = hs2.View.Leader
}

// SyncInfo: basic hotstuff sync information from the selected sync-message
// params:
// - msg: the selected sync-message with sync information
// - leader: the leader of this view
func (hs2 *Hotstuff2) SyncInfo(msg *mgmt.NodeMgmtMsg, leader int) {
	if len(msg.Block) == 0 || len(msg.HsNodes) == 0 {
		fmt.Println("sync message error: message block or hsnode is empty", len(msg.Block) == 0, len(msg.HsNodes) == 0)
		return
	}
	count := len(msg.Block)
	for i := 0; i < count; i++ {
		hs2.LockBlk = append(hs2.LockBlk, &msg.Block[i])
	}
	hs2.PrepareQC = msg.H2Justify
	// hs2.CurHs2Node = msg.HsNodes[0]
	hs2.LockHs2Node = msg.HsNodes[0:]

	hs2.BlkStore.Height = hs2.LockBlk[len(hs2.LockBlk)-1].BlkHdr.Height
	hs2.IgnoreCheckQC = true

	// go to a new round and update
	hs2.NewRound()
	hs2.View.UpdateView(msg.ViewNumber, leader)
}

// UpdateNodesNum: update the node num
// params:
// - nodeNum: the node number need to update
func (hs2 *Hotstuff2) UpdateNodesNum(nodeNum int) {
	hs2.View.NodesNum = nodeNum
}
