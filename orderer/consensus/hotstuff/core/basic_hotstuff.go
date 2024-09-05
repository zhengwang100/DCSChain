/*
Basic HotStuff in XBC

References
----------
Papers: << HotStuff: BFT Consensus with Linearity and Responsiveness >> PODC '19
Links: https://dl.acm.org/doi/pdf/10.1145/3293611.3331591
*/
package core

import (
	"bcrequest"
	"blockchain"
	"bytes"
	common "common"
	"encoding/json"
	"fmt"
	hstypes "hotstuff/types"
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

// BCHotstuff: the core of basic hotstuff consensus
type BCHotstuff struct {
	CurPhase hstypes.StateType // the consensus at which stage
	ConsId   int               // the unique identity in consensus of the node
	View     common.View       // the view consist of view number/leader number/nodes number

	HsNode    common.HsNode // current hotstuff node of this view, which is consist of hash of current block and its parent block
	PrepareQC hstypes.QC    // a quorum certificate storing the highest QC for which a replica voted pre-commit
	LockedQC  hstypes.QC    // a locked quorum certificate storing the highest QC for which a replica voted commit

	LastProposal hstypes.Proposal // last proposal of this view
	CurProposal  hstypes.Proposal // current proposal of this view
	ProposalLock sync.Mutex       // generate proposal lock

	ViewChangeSendFlag bool // the flag that the view-change message should send
	IgnoreCheckQC      bool // the flag ignore the effectiveness of QC

	LastRoundMsg   []*hstypes.Msg // the collection of messages sent by this node in the last view
	CurRoundMsg    []*hstypes.Msg // the collection of messages sent by this node in the current view
	NewViewMsgs    []*hstypes.Msg // the collection of new-view messages this node recieved
	PrepareVotes   []*hstypes.Msg // the collection of prepare vote messages this node recieved
	PreCommitVotes []*hstypes.Msg // the collection of pre-commit vote messages this node recieved
	CommitVotes    []*hstypes.Msg // the collection of commit vote messages this node recieved

	ViewTimer       common.MyTimer         // the timer responsible for liveness
	BlkStore        blockchain.BlockStore  // the unit to generate and store blocks
	ForwardChan     chan []byte            // the channel through which this node receives messages can be responsible for sending messages from the consensus layer to the data layer
	SendChan        chan message.ServerMsg // the channel listened by a node can send messages in the channel to the corresponding node on the network
	Logger          log.Logger             `json:"logger"` // the role of recording logs
	ThresholdSigner *tss.Signer            `json:"Signer"` // the role responsible for threshold signatures
}

// NewBCHotstuff: create an instance of a new consensus of basic hotstuff
// params:
// - timerDuration:	timeout period of the timer
// - consId:		the unique id of this orderer
// - nodeNum:		node number in system
// - path:			the path of block storage
// - senChan:		a channel provided by an upper-layer node through which messages can be sent
// - siger:			signer for threshold sign
// return:
// - a new core of basic hostuff
func NewBCHotstuff(timerDuration int, consId int, nodeNum int, path string, sendChan chan message.ServerMsg, signer *tss.Signer) *BCHotstuff {
	newBCHotstuff := &BCHotstuff{
		ConsId:   consId,
		CurPhase: hstypes.NEW_VIEW,

		View: common.View{
			ViewNumber: 0,
			NodesNum:   nodeNum,
			Leader:     0,
		},
		HsNode: common.HsNode{
			CurHash:    merkle.EmptyHash(),
			ParentHash: merkle.EmptyHash(),
		},
		LockedQC: hstypes.QC{
			QType:      hstypes.PREPARE,
			ViewNumber: -1,
			HsNode: common.HsNode{
				CurHash: merkle.EmptyHash(),
			},
			Sign: nil,
		},
		PrepareQC: hstypes.QC{
			QType:      hstypes.COMMIT,
			ViewNumber: -1,
			HsNode: common.HsNode{
				CurHash: merkle.EmptyHash(),
			},
			Sign: nil,
		},
		BlkStore: blockchain.BlockStore{
			Base:       64, // reserve
			Height:     0,
			Path:       path + "\\r_" + strconv.Itoa(consId),
			PreBlkHash: merkle.EmptyHash(),
			CurBlkHash: merkle.EmptyHash(),
		},
		LastRoundMsg:    []*hstypes.Msg{{ViewNumber: -1}},
		CurRoundMsg:     make([]*hstypes.Msg, 0),
		NewViewMsgs:     make([]*hstypes.Msg, 0),
		PrepareVotes:    make([]*hstypes.Msg, 0),
		PreCommitVotes:  make([]*hstypes.Msg, 0),
		CommitVotes:     make([]*hstypes.Msg, 0),
		ViewTimer:       *common.NewTimer(time.Duration(timerDuration) * time.Millisecond),
		Logger:          *log.New(os.Stdout, "", 0),
		SendChan:        sendChan,
		ThresholdSigner: signer,
	}
	// set log format
	newBCHotstuff.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	return newBCHotstuff
}

// HandleBMsg: the node handle the message to consensus core and send its return message
// params:
// - msgJson: json of basic-hostuff message
func (bhs *BCHotstuff) HandleBMsg(msgJson []byte, pk []byte) {
	// convert json to basic-hotstuff message
	var msg hstypes.Msg
	err := json.Unmarshal(msgJson, &msg)
	if err != nil {
		bhs.Logger.Println("[ERROR]:", bhs.GetNodeName(), err)
		return
	}
	// submit the message to basic hotstuff and get its return messages
	msgReturn := bhs.RouteBMsg(&msg, pk)
	if msgReturn == nil {
		return
	}

	// if return message is not nil, sent it
	msgReturn.SendNode = bhs.GetNodeName()
	bhs.CurRoundMsg = append(bhs.CurRoundMsg, msgReturn)
	bhs.SendSerMsg(msgReturn)

	// execute cmds and store the proposal into local blockchain
	if msgReturn.MType == hstypes.NEW_VIEW && !bhs.CurProposal.IsEmpty() {

		// if node successfully execute it, store it to blockchain
		if bhs.Execute() {
			go bhs.SendSerMsg(&hstypes.Msg{
				ViewNumber: bhs.View.ViewNumber - 1,
				HsNode:     bhs.HsNode,
				Proposal:   hstypes.Proposal{},
				SendNode:   bhs.GetNodeName(),
				ReciNode:   "Client",
			})

			// bhs.Logger.Println("[EXECUTE]", bhs.GetNodeName()+" Success!", bhs.View.ViewNumber-1)
			// n.NodeManager.MsgLog = n.BasicHotstuff.LastRoundMsg
			bhs.UpdateBasicHotstuff()
		}
	}

}

// RouteBMsg: the node in basic hotstuff choose conresponding func to handle the message by message type
// params:
// - recieved message
// return:
// - message waiting to be sent
func (bhs *BCHotstuff) RouteBMsg(msg *hstypes.Msg, pk []byte) *hstypes.Msg {
	switch msg.MType {
	case 0:
		return bhs.HandleNewView(msg)
	case 1:
		return bhs.HandlePrepare(msg, pk)
	case 2:
		return bhs.HandlePrepareVote(msg)
	case 3:
		return bhs.HandlePreCommit(msg)
	case 4:
		return bhs.HandlePreCommitVote(msg)
	case 5:
		return bhs.HandleCommit(msg)
	case 6:
		return bhs.HandleCommitVote(msg)
	case 7:
		return bhs.HandleDecide(msg)
	default:
		fmt.Println("The type of this message isn't included konw type", bhs.GetNodeName(), msg)
		return nil
	}
}

// HandleReq: the basic-hotstuff handle the request
// params:
// - height: 	block height
// - preHash: 	hash of previous block
// - req: 		recieved requests
func (bhs *BCHotstuff) HandleReq(height int, preHash []byte, req []bcrequest.BCRequest) {
	bhs.ProposalLock.Lock()
	defer bhs.ProposalLock.Unlock()

	// generate a new proposal
	bhs.CurProposal = hstypes.Proposal{
		Height:     height,
		PreBlkHash: preHash,
		Commands:   make([][]byte, 0),
		Signs:      make([][]byte, 0),
	}
	// fmt.Println(len(req), bhs.View.ViewNumber)

	for i := 0; i < len(req); i++ {
		bhs.CurProposal.Commands = append(bhs.CurProposal.Commands, req[i].Cmd)
		bhs.CurProposal.Signs = append(bhs.CurProposal.Signs, req[i].Sign)
	}

	bhs.CurProposal.RootHash = merkle.HashFromByteSlicesIterative(bhs.CurProposal.Commands)

	// newMsg := hstypes.Msg{MType: hstypes.NEW_VIEW, ReciNode: bhs.GetNodeName()}
	// bhs.SendSerMsg(&newMsg)

	// if the node is waiting the requests, process the prepare phase of leader

	if bhs.CurPhase == hstypes.WAITING {
		msgReturn := bhs.GenProposal()
		if msgReturn == nil {
			return
		}
		// send the prepare message
		bhs.CurRoundMsg = append(bhs.CurRoundMsg, msgReturn)
		bhs.SendSerMsg(msgReturn)
	}
}

// SendSerMsg: send the message, in fact the chan provided by the outer layer is passed to the outer layer,
// and the outer layer sends the message
// params:
// - msg: message that need to be sent
func (bhs *BCHotstuff) SendSerMsg(msg *hstypes.Msg) {
	msgJson, err := json.Marshal(msg)
	if err != nil {
		return
	}
	serMsg := message.ServerMsg{
		SType:      message.ORDER,
		SendServer: msg.SendNode,
		ReciServer: msg.ReciNode,
		Payload:    msgJson,
	}
	bhs.SendChan <- serMsg
}

// IsLeader: check whether self is leader
// note: for simplicity, use the view mode directly to summarize the nodes and the node takes turns to be the leader
// furthermore, if the leader off-line, add a number and change the turn
// return:
// - true if the leader, false otherwise
func (bhs *BCHotstuff) IsLeader() bool {
	return bhs.GetNodeName() == bhs.View.LeaderName()
	// return bhs.GetLeaderName() == bhs.GetLeaderName()
}

// SafeNode: check whether the node is safe in basic hotstuff
// SafeNode implement hotstuff description as follow:
// function safeNode(node, qc)
//
//	return (node extends from lockedQC.node) ∨ (qc.viewNumber > lockedQC.viewNumber) // safe rule ∨ liveness rule
//
// params:
// - hsNode: node to be checked
// - qc: qc used for node verification
// return:
// - true if node is safenode, false otherwise
func (bhs *BCHotstuff) SafeNode(hsNode *common.HsNode, qc *hstypes.QC) bool {
	// fmt.Println("SafeNode", bhs.LockedQC.HsNode.CurHash, hsNode.ParentHash, qc.ViewNumber > bhs.LockedQC.ViewNumber)
	return bytes.Equal(bhs.LockedQC.HsNode.CurHash, hsNode.ParentHash) || qc.ViewNumber > bhs.LockedQC.ViewNumber
}

// CheckQC: check message's QC is valid, include QC's type, view and signature
// mathc means that the view and type of qc meet the requirements
// validity means that the threshold signature of QC is valid and can be verified
// params:
// - msg: 			need to check QC message
// - code:			type of QC should be
// - viewNumber:	view of QC should be generated in
// return:
// - true if both match and validity pass, false otherwise
func (bhs *BCHotstuff) CheckQC(msg *hstypes.Msg, code hstypes.StateType, viewNumber int) bool {

	// check the match of the QC carried by the message
	if !bhs.MatchingQC(&msg.Justify, code, viewNumber) {
		fmt.Println("MatchingQC", bhs.GetNodeName(), bhs.CurPhase, msg.Justify.QType, code, msg.Justify.ViewNumber, viewNumber)
		return false
	}

	// check the validity of the QC carried by the message
	if !bhs.ThresholdSigner.ThresholdSignVerify(msg.Justify.QC2SignMsgByte(), msg.Justify.Sign) {
		fmt.Println("!bhs.ThresholdSigner", bhs.GetNodeName())
		return false
	}

	// return true if both match and validity pass
	return true
}

// CombineSig: combine message's part signature to a complete signature
// params:
// - voteMsgs: the silce of recieved messages with part signature
// ruturn:
// - byte silce of the complete signature
func (bhs *BCHotstuff) CombineSign(voteMsgs *[]*hstypes.Msg) []byte {

	// declare a two-dimensional byte slices for collecting part signatures
	var partSigs [][]byte
	for _, m := range *voteMsgs {
		partSigs = append(partSigs, m.PartialSig)
		// fmt.Println("Combinsign", bhs.CurPhase, m.SendNode, m.PartialSig)
	}

	// recover the signed message
	// msgSign := (*(bhs.CurRoundMsg)[len(bhs.CurRoundMsg)-1])
	msgSign := hstypes.Msg{
		MType:      bhs.CurPhase,
		ViewNumber: bhs.View.ViewNumber,
	}
	msgSign.HsNode = bhs.HsNode

	// combine the complete signature according to part signature and recovered messages
	sig, err := bhs.ThresholdSigner.CombineSig(msgSign.Message2Byte(), partSigs)
	if err == nil {
		return sig
	}
	return nil
}

// CreateLeaf: generate a new node extend from parent
// params:
// -parent: byte slice for parent which is last proposal hash
// return:
// - new node
func (bhs *BCHotstuff) CreateLeaf(parent []byte) common.HsNode {
	return common.HsNode{
		ParentHash: parent,
		CurHash:    bhs.BlkStore.CurBlkHash,
	}
}

// Log: the basic hotstuff log
func (bhs *BCHotstuff) Log() {
	switch bhs.CurPhase {
	case hstypes.NEW_VIEW:
		bhs.Logger.Println("[NEW_VIEW]", "r_"+strconv.Itoa(bhs.ConsId)+" Success!")
	}
}

// GetNodeName: get self node name
func (bhs *BCHotstuff) GetNodeName() string {
	return "r_" + strconv.Itoa(bhs.ConsId)
}

// GetLeaderName: get leader of current view name
func (bhs *BCHotstuff) GetLeaderName() string {
	return bhs.View.LeaderName()
}

// MatchingMsg: check whether message's type and view are matching
// params:
// - msg: 			message need to check
// - code:			type of message should be
// - viewNumber:	view of message should be generated in
// return:
// - true if match pass, false otherwise
func (bhs *BCHotstuff) MatchingMsg(msg *hstypes.Msg, code hstypes.StateType, curView int) bool {
	if msg.MType == code && msg.ViewNumber == curView {
		return true
	}
	// fmt.Println(msg.MType, code, msg.ViewNumber, curView)
	return false
}

// MatchingQC: check QC's type and view
// params:
// - qc: 			QC need to check
// - code:			type of QC should be
// - viewNumber:	view of QC should be generated in
// return:
// - true if match pass, false otherwise
func (bhs *BCHotstuff) MatchingQC(qc *hstypes.QC, code hstypes.StateType, curView int) bool {
	if qc.QType == code && qc.ViewNumber == curView {
		return true
	}
	return false
}

// UpdateNode: refresh the node in this system
func (bhs *BCHotstuff) UpdateBasicHotstuff() {
	bhs.CurPhase = hstypes.NEW_VIEW
	bhs.LastProposal = bhs.CurProposal
	bhs.CurProposal = hstypes.Proposal{}
}

// UpdateNodesNum: update the node num
// params:
// - nodeNum: the node number need to update
func (bhs *BCHotstuff) UpdateNodesNum(nodeNum int) {
	bhs.View.NodesNum = nodeNum
}

// ClearCurrentRound: clears the message for the current view,
// but the leader keeps the message for that view and the new-view message for liveness
func (bhs *BCHotstuff) ClearCurrentRound() {
	if bhs.GetNodeName() != bhs.GetLeaderName() {
		// replica clear the proposal and new-view messages
		bhs.CurProposal = hstypes.Proposal{}
		bhs.NewViewMsgs = make([]*hstypes.Msg, 0)
	}

	// update the other messages recieved in this view
	bhs.CurRoundMsg = make([]*hstypes.Msg, 0)
	bhs.PrepareVotes = make([]*hstypes.Msg, 0)
	bhs.PreCommitVotes = make([]*hstypes.Msg, 0)
	bhs.CommitVotes = make([]*hstypes.Msg, 0)

	// generate an empty block
	bhs.BlkStore.GenEmptyBlock()

	// update the current phase
	bhs.CurPhase = hstypes.NEW_VIEW
}

// AddSyncInfo: add local information to message for sync, include view number/current block/hotstuff node/prepareQC/leader
// params:
// - msg: message which sync information needs to be added
func (bhs *BCHotstuff) AddSyncInfo(msg *mgmt.NodeMgmtMsg) {
	msg.ViewNumber = bhs.View.ViewNumber
	msg.Block = append(msg.Block, bhs.BlkStore.CurProposalBlk)
	msg.HsNodes = append(msg.HsNodes, bhs.HsNode)
	msg.Justify = bhs.PrepareQC
	msg.Leader = bhs.View.Leader
}

// SyncInfo: basic hotstuff sync information from the selected sync-message
// params:
// - msg: the selected sync-message with sync information
// - leader: the leader of this view
func (bhs *BCHotstuff) SyncInfo(msg *mgmt.NodeMgmtMsg, leader int) {
	bhs.BlkStore.CurProposalBlk = msg.Block[0]
	bhs.PrepareQC = msg.Justify
	bhs.HsNode = msg.HsNodes[0]
	bhs.BlkStore.Height = bhs.BlkStore.CurProposalBlk.BlkData.Height

	// store the local block recieved
	bhs.BlkStore.StoreBlock(bhs.BlkStore.CurProposalBlk)
	// go to a new round and update
	bhs.NewRound()
	bhs.View.UpdateView(msg.ViewNumber, leader)
}

// RestartBasicHotstuff: restart the basic hotstuff to handle the request and message
// return:
// - the prepare message
func (bhs *BCHotstuff) RestartBasicHotstuff() *hstypes.Msg {
	if bhs.GetLeaderName() == bhs.GetNodeName() {
		if !bhs.CurProposal.IsEmpty() {
			msgReturn := bhs.GenProposal()

			if msgReturn != nil {
				bhs.CurRoundMsg = append(bhs.CurRoundMsg, msgReturn)
			}
			return msgReturn
		} else {
			bhs.CurPhase = hstypes.WAITING
			return nil
		}
	}
	return nil
}

// initLeader: the node which is leader in view init
func (bhs *BCHotstuff) InitLeader() {

	// leader phase is waiting and ignore check QC
	bhs.CurPhase = hstypes.WAITING
	bhs.IgnoreCheckQC = true

	if bhs.View.ViewNumber == 0 {
		// in view 0, add an empty new-view message to self
		msg := hstypes.Msg{
			MType:      hstypes.NEW_VIEW,
			ViewNumber: 0,
			SendNode:   "r_" + strconv.Itoa(0),
			HsNode: common.HsNode{
				CurHash:    merkle.EmptyHash(),
				ParentHash: merkle.EmptyHash(),
			},
			Justify:    hstypes.QC{},
			PartialSig: nil,
		}
		bhs.NewViewMsgs = append(bhs.NewViewMsgs, &msg)
	} else {
		// add a message rely on local state to self
		msg := hstypes.Msg{
			MType:      hstypes.NEW_VIEW,
			ViewNumber: bhs.View.ViewNumber - 1,
			SendNode:   bhs.GetNodeName(),
			HsNode: common.HsNode{
				CurHash:    merkle.EmptyHash(),
				ParentHash: merkle.EmptyHash(),
			},
			Justify:    bhs.PrepareQC,
			PartialSig: nil,
		}
		bhs.NewViewMsgs = append(bhs.NewViewMsgs, &msg)
	}
}

// FixLeader: adds an empty new view message to the leader
func (bhs *BCHotstuff) FixLeader() {

	count := (bhs.View.NodesNum-1)/3*2 - len(bhs.NewViewMsgs) + 1
	for i := 0; i < count; i++ {
		// add an empty new-view message to self for liveness
		msg := hstypes.Msg{
			MType:      hstypes.NEW_VIEW,
			ViewNumber: 0,
			SendNode:   "r_" + strconv.Itoa(0),
			HsNode: common.HsNode{
				CurHash:    merkle.EmptyHash(),
				ParentHash: merkle.EmptyHash(),
			},
			Justify:    hstypes.QC{},
			PartialSig: nil,
		}
		bhs.NewViewMsgs = append(bhs.NewViewMsgs, &msg)
	}
}

// Execute: execute the commands
func (bhs *BCHotstuff) Execute() bool {
	return true
}

func (bhs *BCHotstuff) VerifyReqs(reqs [][]byte, signs [][]byte, pk []byte) bool {
	length := len(reqs)
	if length == 0 {
		bhs.Logger.Println("[Error]: requests length is zero", bhs.GetNodeName(), bhs.View.ViewNumber, bhs.CurPhase)
		return false
	}
	// for i := 0; i < length; i++ {
	// 	if !sm2.Sm2Verify(signs[i], pk, reqs[i]) {
	// 		bhs.Logger.Println("[Error]: requests sign verify error")
	// 		return false
	// 	}
	// }
	return true
}
