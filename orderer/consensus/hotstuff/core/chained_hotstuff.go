/*
Chained HotStuff in XBC

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

// CHotstuff: the core of chained hotstuff consensus
type CHotstuff struct {
	CurPhase        hstypes.StateType // the consensus at which stage
	View            common.View       // the view consist of view number/leader number/nodes number
	ConsId          int               // the unique identity in consensus of the node
	ExecuteState    bool              // the flag of the last locked block can be executed
	LastLeaderState bool              // the flag indicating whether a new-view message from the previous view leader was received

	// b*, b", b', b
	HsNodes   [4]common.HsNode    // 4 current hotstuff nodes of this view, which each is consist of hash of current block and its parent block
	Blocks    [4]blockchain.Block // 4 current blocks of this view, which went through the stages of prepare, pre-commit, commit and decide respectively
	GenericQC hstypes.ChainedQC   // the same as PrepareQC in basic hotstuff, a quorum certificate storing the highest QC for which a replica voted pre-commit
	LockedQC  hstypes.ChainedQC   // the same as LockedQC in basic hotstuff, a locked quorum certificate storing the highest QC for which a replica voted commit

	LastProposal hstypes.Proposal // last proposal of this view
	CurProposal  hstypes.Proposal // current proposal of this view
	ProposalLock sync.Mutex

	CurRoundMsg     *hstypes.CMsg   // the generic message sent by this node in the current view
	NewViewMsgs     []*hstypes.CMsg // the collection of new-view messages this node recieved
	GenericVoteMsgs []*hstypes.CMsg // the collection of generic vote messages this node recieved

	ViewChangeSendFlag bool // the flag that the view-change message should send
	ViewChangeFlag     bool // the flag that is in the view-change phase

	ViewTimer       common.MyTimer         // the timer responsible for liveness
	BlkStore        blockchain.BlockStore  // generate and store blocks
	ForwardChan     chan []byte            // the channel through which this node receives messages can be responsible for sending messages from the consensus layer to the data layer
	SendChan        chan message.ServerMsg // the channel listened by a node can send messages in the channel to the corresponding node on the network
	Logger          log.Logger             `json:"logger"` // the role of recording logs
	ThresholdSigner *tss.Signer            `json:"Signer"` // the role responsible for threshold signatures
}

// NewChainedHotstuff: create an instance of a new consensus of chained hotstuff
// params:
// - timerDuration:	timeout period of the timer
// - consId:		the unique id of this orderer
// - nodeNum:		node number in system
// - path:			the path of block storage
// - senChan:		a channel provided by an upper-layer node through which messages can be sent
// - siger:			signer for threshold sign
// return:
// - a new core of chained hostuff
func NewChainedHotstuff(timerDuration int, consId int, nodeNum int, path string, sendChan chan message.ServerMsg, signer *tss.Signer) *CHotstuff {
	// declare hotstuff nodes and new empty proposal
	var hsNodes [4]common.HsNode
	proposal := hstypes.Proposal{}
	emptyHash := proposal.GenProposalHash()

	for i := range hsNodes {
		hsNodes[i] = common.HsNode{
			CurHash:    emptyHash,
			ParentHash: emptyHash,
		}
	}
	newChainedHotstuff := CHotstuff{
		ConsId:   consId,
		CurPhase: hstypes.NEW_VIEW,
		View: common.View{
			ViewNumber: 0,
			NodesNum:   nodeNum,
			Leader:     0,
		},
		ExecuteState: false,
		HsNodes:      hsNodes,
		GenericQC: hstypes.ChainedQC{
			ViewNumber: -1,
			HsNodes:    hsNodes,
		},
		LockedQC: hstypes.ChainedQC{
			ViewNumber: -1,
			HsNodes:    hsNodes,
		},
		BlkStore: blockchain.BlockStore{
			Base:            64,
			Height:          0,
			GeneratedHeight: 0,
			Path:            path + "\\r_" + strconv.Itoa(consId),
		},
		ViewTimer:       *common.NewTimer(time.Duration(timerDuration) * time.Millisecond),
		Logger:          *log.New(os.Stdout, "", 0),
		SendChan:        sendChan,
		ThresholdSigner: signer,
	}

	// set log format
	newChainedHotstuff.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	return &newChainedHotstuff
}

// HandleCMsg: the node handle the message to consensus core and send its return message
// params:
// - msgJson: json of chained-hostuff message
func (chs *CHotstuff) HandleCMsg(msgJson []byte) {
	// convert json to message
	var msg hstypes.CMsg
	err := json.Unmarshal(msgJson, &msg)
	if err != nil {
		chs.Logger.Println("[ERROR]:", chs.GetNodeName(), err)
		return
	}

	// submit the chained message to chained hotstuff and get its return messages
	msgReturnSlice := chs.RouteCMsg(&msg)

	// send its return messages and execute
	if len(msgReturnSlice) == 0 {
		return
	}

	for _, msgReturn := range msgReturnSlice {
		if msgReturn == nil {
			continue
		}

		// add node name and record it
		msgReturn.SendNode = chs.GetNodeName()
		if msgReturn.MType == hstypes.GENERIC {
			chs.CurRoundMsg = msgReturn
		}
		go chs.SendSerMsg(msgReturn)

		// execute cmds and store the proposal into local blockchain
		if msgReturn.MType == hstypes.NEW_VIEW && chs.ExecuteState && len(chs.Blocks[3].BlkHdr.Validation) != 0 {

			// if node successfully execute it, store it to blockchain
			if chs.Execute() {
				go chs.SendSerMsg(&hstypes.CMsg{
					ViewNumber: chs.View.ViewNumber - 1,
					Proposal:   hstypes.Proposal{},
					SendNode:   chs.GetNodeName(),
					ReciNode:   "Client",
				})

				// fmt.Println(time.Now())
				// chs.Logger.Println("[EXECUTE]", chs.GetNodeName(), " Success!", chs.View.ViewNumber)
				// update the chained hotstuff execute state
				chs.ExecuteState = false
			}
		}
	}
}

// RouteCMsg: the node in basic hotstuff choose conresponding func to handle the message by message type
// params:
// - msg: recieved chained message
// return:
// - message waiting to be sent
func (chs *CHotstuff) RouteCMsg(msg *hstypes.CMsg) []*hstypes.CMsg {
	switch msg.MType {
	case hstypes.NEW_VIEW:
		return chs.CHandleNewView(msg)
	case hstypes.GENERIC:
		return chs.CHandleGeneric(msg)
	case hstypes.GENERIC_VOTE:
		return chs.CHandleGenericVote(msg)
	default:
		fmt.Println("The type of this message isn't included konw type")
		return nil
	}
}

// HandleReq: the chained-hotstuff handle the request
// params:
// - height: 	block height
// - preHash: 	hash of previous block
// - req: 		recieved requests
func (chs *CHotstuff) HandleReq(height int, preHash []byte, req []bcrequest.BCRequest) {
	if chs.CurPhase == hstypes.WAITING {
		chs.ProposalLock.Lock()
		defer chs.ProposalLock.Unlock()

		chs.CurProposal = hstypes.Proposal{
			Height:     height,
			PreBlkHash: preHash,
			Commands:   make([][]byte, 0),
		}
		for i := 0; i < len(req); i++ {
			chs.CurProposal.Commands = append(chs.CurProposal.Commands, req[i].Cmd)
		}
		chs.CurProposal.RootHash = merkle.HashFromByteSlicesIterative(chs.CurProposal.Commands)

		if chs.CurPhase == hstypes.WAITING {
			// chs.CurPhase = hstypes.NEW_VIEW
			msgReturn := chs.GenProposal()
			if msgReturn != nil {
				chs.CurRoundMsg = msgReturn
				chs.SendSerMsg(msgReturn)
			}
		}
	}
}

// RestartChainedHotstuff: restart the chained hotstuff to handle the request and message
// return:
// - the prepare message
func (chs *CHotstuff) RestartChainedHotstuff() *hstypes.CMsg {
	if chs.GetLeaderName() == chs.GetNodeName() {
		if !chs.CurProposal.IsEmpty() {
			msgReturn := chs.GenProposal()

			if msgReturn != nil {
				chs.CurRoundMsg = msgReturn
			}
			return msgReturn
		} else {
			chs.CurPhase = hstypes.WAITING
			return nil
		}
	}
	return nil
}

// IsLeader: check whether self is leader
func (chs *CHotstuff) IsLeader() bool {
	return chs.GetNodeName() == chs.View.LeaderName()
}

// UpdateHsNode: update local hotstuff nodes with new node, move one digit backwards
func (chs *CHotstuff) UpdateHsNode(newCHSNode *common.HsNode) {
	chs.HsNodes[3] = chs.HsNodes[2]
	chs.HsNodes[2] = chs.HsNodes[1]
	chs.HsNodes[1] = chs.HsNodes[0]
	chs.HsNodes[0] = *newCHSNode
}

// UpdateProposal: update local proposals with new proposal, move one digit backwards
func (chs *CHotstuff) UpdateBlock(blk *blockchain.Block) {
	chs.Blocks[3] = chs.Blocks[2]
	chs.Blocks[2] = chs.Blocks[1]
	chs.Blocks[1] = chs.Blocks[0]
	chs.Blocks[0] = *blk
	if len(blk.BlkData.Trans) == 0 {
		chs.BlkStore.CurBlkHash = merkle.EmptyHash()
	}
	chs.BlkStore.GeneratedHeight = blk.BlkHdr.Height + 1
}

// CreateLeaf: generate a new node extend from parent
// params: byte slice for parent which is last proposal hash
// return: new node
// note: this func is same as the same-named func in basic_hotstuff.go, but are bind to different classes
func (chs *CHotstuff) CreateLeaf(parent []byte) common.HsNode {
	// fmt.Println("parent", parent)
	return common.HsNode{
		ParentHash: parent,
		CurHash:    chs.BlkStore.CurBlkHash,
	}
}

// CheckCMsg: check chained message according message's type
func CheckCMsg(msg *hstypes.CMsg, code hstypes.StateType, viewNum int) bool {
	switch code {
	case hstypes.NEW_VIEW:
		return MatchingCMsg(msg, code, viewNum-1)
	case hstypes.GENERIC_VOTE:
		return MatchingCMsg(msg, code, viewNum)
	default:
		return false
	}
}

// MatchingCMsg: check whether chianed message's type and view are matching
func MatchingCMsg(msg *hstypes.CMsg, code hstypes.StateType, curView int) bool {
	// fmt.Println(msg.MType, msg.MType == code, msg.ViewNumber, curView, msg.ViewNumber == curView)
	return msg.MType == code && msg.ViewNumber == curView
}

// MatchingCQC: check chained QC's type and view
// params: qurom certfication, qc's type, current view
// return: result
func MatchingCQC(qc *hstypes.QC, code hstypes.StateType, curView int) bool {
	if qc.QType == code && qc.ViewNumber == curView {
		return true
	}
	return false
}

// CombineSign: combine part signatures to a complete signature
// params: the silce of recieved messages with part signature
// ruturn: byte silce of the complete signature
func (chs *CHotstuff) CombineSign(voteMsgs []*hstypes.CMsg) []byte {
	// declare a two-dimensional byte slices for collecting part signatures
	var partSigs [][]byte
	for _, m := range voteMsgs {
		partSigs = append(partSigs, m.PartialSig)
		// fmt.Println("Combinsign", chs.CurPhase, m.SendNode, m.PartialSig)
	}

	// combine the complete signature according to part signature and recovered messages
	sig, err := chs.ThresholdSigner.CombineSig(chs.CurRoundMsg.ChainedMessage2Byte(), partSigs)
	// sig, err := chs.ThresholdSigner.CombineSig(msgSign.ChainedMessage2Byte(), partSigs)
	if err == nil {
		return sig
	}
	return nil
}

// SafeNode: check whether the node is safe in chained hotstuff
func (chs *CHotstuff) SafeNode(hsNode [4]common.HsNode, qc *hstypes.ChainedQC) bool {
	// fmt.Println("SafeNode", qc.HsNodes[0].CurHash, hsNode[0].ParentHash, bytes.Equal(qc.HsNodes[0].CurHash, hsNode[0].ParentHash), qc.ViewNumber, chs.LockedQC.ViewNumber)
	return bytes.Equal(qc.HsNodes[0].CurHash, hsNode[0].ParentHash) || qc.ViewNumber > chs.LockedQC.ViewNumber
}

// GetChainedLastLeader: get the current leader of this view
func (chs *CHotstuff) GetChainedCurLeader() string {
	return chs.View.LeaderName()
}

// GetChainedLastLeader: get the last leader of this view
func (chs *CHotstuff) GetChainedLastLeader() string {
	return chs.View.LastLeaderName()
}

// ExistNotExecuteBlock: check whether there's a non-empty block in node current proposals
func (chs *CHotstuff) ExistNotExecuteBlock() bool {
	// for i := range chs.Blocks[:len(chs.Blocks)-1] {
	// 	if len(chs.Blocks[i].BlkData.Trans) != 0 {
	// 		return true
	// 	}
	// }
	return false
}

// initLeader: the node which is leader in view init
func (chs *CHotstuff) InitLeader() {
	// generate empty proposal to init
	proposal := hstypes.Proposal{}
	emptyHash := proposal.GenProposalHash()
	hsNodes := [4]common.HsNode{{
		CurHash:    emptyHash,
		ParentHash: emptyHash,
	}}
	msg := hstypes.CMsg{
		MType:      hstypes.NEW_VIEW,
		ViewNumber: 0,
		SendNode:   "r_0",
		ReciNode:   "r_0",
		HsNodes:    hsNodes,
		Justify: hstypes.ChainedQC{
			QType:      hstypes.NEW_VIEW,
			ViewNumber: -1,
			HsNodes:    hsNodes,
			Sign:       nil,
		},
		PartialSig: nil,
	}
	count := (chs.View.NodesNum - 1) / 3 * 2
	for i := 0; i < count; i++ {
		chs.NewViewMsgs = append(chs.NewViewMsgs, &msg)
	}
	chs.CurPhase = hstypes.WAITING
}

// FixLeader: adds an empty new view message to the leader
func (chs *CHotstuff) FixLeader() {

	count := (chs.View.NodesNum-1)/3*2 - len(chs.NewViewMsgs) + 1

	proposal := hstypes.Proposal{}
	emptyHash := proposal.GenProposalHash()
	hsNodes := [4]common.HsNode{{
		CurHash:    emptyHash,
		ParentHash: emptyHash,
	}}
	for i := 0; i < count; i++ {
		// add an empty new-view message to self for liveness
		msg := hstypes.CMsg{
			MType:      hstypes.NEW_VIEW,
			ViewNumber: 0,
			SendNode:   "r_0",
			ReciNode:   "r_0",
			HsNodes:    hsNodes,
			Justify: hstypes.ChainedQC{
				QType:      hstypes.NEW_VIEW,
				ViewNumber: -1,
				HsNodes:    hsNodes,
				Sign:       nil,
			},
			PartialSig: nil,
		}
		chs.NewViewMsgs = append(chs.NewViewMsgs, &msg)
	}
}

// SendSerMsg: send the message, in fact the chan provided by the outer layer is passed to the outer layer,
// and the outer layer sends the message
// params:
// - msg: message that need to be sent
func (chs *CHotstuff) SendSerMsg(msg *hstypes.CMsg) {
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

	chs.SendChan <- serMsg
}

// Execute: execute the commands
func (chs *CHotstuff) Execute() bool {
	return true
}

// ClearCurrentRound: clears the message for the current view,
// but the leader keeps the message for that view and the new-view message for liveness
func (chs *CHotstuff) ClearCurrentRound() {
	if chs.GetNodeName() != chs.GetLeaderName() {
		// replica clear the proposal and new-view messages
		chs.CurProposal = hstypes.Proposal{}
		chs.NewViewMsgs = make([]*hstypes.CMsg, 0)
	}

	// update the other messages recieved in this view
	chs.CurRoundMsg = &hstypes.CMsg{}

	// generate an empty block
	chs.BlkStore.GenEmptyBlock()

	// update the current phase
	chs.CurPhase = hstypes.NEW_VIEW
}

// AddSyncInfo: add local information to message for sync, include view number/current block/hotstuff node/prepareQC/leader
// params:
// - msg: message which sync information needs to be added
func (chs *CHotstuff) AddSyncInfo(msg *mgmt.NodeMgmtMsg) {
	msg.ViewNumber = chs.View.ViewNumber
	msg.Block = append(msg.Block, chs.Blocks[0], chs.Blocks[1], chs.Blocks[2], chs.Blocks[3])
	msg.HsNodes = append(msg.HsNodes, chs.HsNodes[0], chs.HsNodes[1], chs.HsNodes[2], chs.HsNodes[3])
	msg.CJustify = chs.GenericQC
	msg.Leader = chs.View.Leader
}

// UpdateNodesNum: update the node num
// params:
// - nodeNum: the node number need to update
func (chs *CHotstuff) UpdateNodesNum(nodeNum int) {
	chs.View.NodesNum = nodeNum
}

// CSyncInfo: basic hotstuff sync information from the selected sync-message
// params:
// - msg: the selected sync-message with sync information
// - leader: the leader of this view
func (chs *CHotstuff) CSyncInfo(msg *hstypes.CMsg, leader int) {
	chs.BlkStore.CurProposalBlk = msg.Blk
	chs.GenericQC = msg.Justify
	chs.HsNodes = msg.HsNodes
	chs.BlkStore.Height = chs.CurProposal.Height

	// store the local block recieved
	chs.BlkStore.StoreBlock(chs.BlkStore.CurProposalBlk)

	// go to a new round and update
	chs.NewRound()
	chs.View.UpdateView(msg.ViewNumber, leader)
}

// NewRound: start a new round consensus, refresh the consensus state
func (chs *CHotstuff) NewRound() {
	// chs.CurPhase = hstypes.NEW_VIEW
	chs.View.NextView()
	chs.CurRoundMsg = &hstypes.CMsg{}
	chs.NewViewMsgs = make([]*hstypes.CMsg, 0)
	chs.GenericVoteMsgs = make([]*hstypes.CMsg, 0)
}

// SyncInfo: chained hotstuff sync information from the selected sync-message
// params:
// - msg: the selected sync-message with sync information
// - leader: the leader of this view
func (chs *CHotstuff) SyncInfo(msg *mgmt.NodeMgmtMsg, leader int) {
	if len(msg.Block) != 4 || len(msg.HsNodes) != 4 {
		fmt.Println("sync message error: message block or hsnode number is error", len(msg.Block), len(msg.HsNodes))
		return
	}
	for i := 0; i < 4; i++ {
		chs.Blocks[i] = msg.Block[i]
		chs.HsNodes[i] = msg.HsNodes[i]
	}
	chs.GenericQC = msg.CJustify
	// go to a new round and update
	chs.NewRound()
	chs.View.UpdateView(msg.ViewNumber, leader)
}
