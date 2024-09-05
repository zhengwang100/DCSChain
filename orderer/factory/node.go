package ofactory

import (
	"bcmanager"
	"blockchain"
	"common"
	"crypto/rand"
	"fmt"
	"hotstuff/core"
	hstypes "hotstuff/types"
	h2core "hotstuff2/core"
	"mgmt"
	pcore "pbft/core"
	"strconv"
	"time"

	"github.com/xlcetc/cryptogm/sm/sm2"
)

// NodeMsgBufferLength
const NodeMsgBufferLength uint8 = 8

// Node is the system node which is the main unit
// Node contents:
// NodeID:					the only identity the node in system
// HandleState:				the flag of indicating whether messages are received from the channel
// ReqState:				the flag of indicating whether the request is received from the channel
//
// BasicHotstuff: 			the basic hotstuff core
// ChaindeConsensus: 		the chained hotstuff core
// Hotstuff2: 			the basic hotstuff-2 core
// PBFTConsensus: 			the PBFT core
//
// NodeManager:				the basic node manager
//
// BlkStore: 				related to blockchain store
// Requests: 				the node recieved request from clients
type Node struct {
	NodeID      common.PrivID
	HandleState bool
	ReqState    bool

	ConsType        common.ConsensusType
	Cons            interface{}
	BasicHotstuff   *core.BCHotstuff
	ChainedHotstuff *core.CHotstuff
	Hotstuff2       *h2core.Hotstuff2
	PBFTConsensus   *pcore.PBFT

	NMType      mgmt.NodeManagerType
	NodeManager bcmanager.NodeManager
	// NodeManager   bhmanager.NodeManager

	BlkStore blockchain.BlockStore
	Requests [][]byte
}

// NewNode: generate a new node and init according to name and consensus protocol type
// params: name(node's name),consType(consensus protocol type)
// return: node(new node)
func NewNode(id int, consType common.ConsensusType, path string) (*Node, error) {
	name := "r_" + strconv.Itoa(id)
	// generate node private key and public key
	sk, pk, _ := sm2.Sm2KeyGen(rand.Reader)

	// init nodesTable and add self
	nodesTable := map[string]mgmt.NodeKey{
		name: {
			Name:      name,
			Sm2PubKey: pk,
		},
	}

	// new node
	var req [][]byte
	node := &Node{
		NodeID: common.PrivID{
			ID: common.PubID{
				Name:   name,
				PubKey: pk,
			},
			Address:    make(chan []byte, 64),
			PrivateKey: sk,
		},
		ConsType:    consType,
		HandleState: true,
		ReqState:    true,
		Requests:    req,
	}

	// init node manager
	node.InitNodeManager(mgmt.BASIC, id, nodesTable)

	// init consensus
	node.InitConsensus(consType, path, id)

	return node, nil
}

// InitConsensus: init consensus, until now only basic hotstuff and chained hotstuff
func (n *Node) InitConsensus(consType common.ConsensusType, path string, id int) {
	switch consType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		n.BasicHotstuff = core.NewBCHotstuff(2000, id, 0, path, nil, nil)
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		n.ChainedHotstuff = core.NewChainedHotstuff(2000, id, 0, path, nil, nil)
	case common.HOTSTUFF_2_PROTOCOL:
		n.Hotstuff2 = h2core.NewHotstuff2(500, 2000, id, 0, path, nil, nil)
	case common.PBFT:
		n.PBFTConsensus = pcore.NewPBFT(2000, id, 0, path, nil, nil)
	default:
		fmt.Println("Consensus type is unknown type!")
	}
}

// InitConsensus: init consensus, until now only basic hotstuff and chained hotstuff
func (n *Node) InitNodeManager(nmType mgmt.NodeManagerType, id int, nodesTable map[string]mgmt.NodeKey) {

	switch nmType {
	case mgmt.BASIC:
		n.NodeManager = *bcmanager.NewNodeManager(id, nodesTable, map[string]chan []byte{})
	default:
		fmt.Println("NodeManager type is unknown type!")
	}
}

func (n *Node) GetBlockStore() *blockchain.BlockStore {
	return &n.BlkStore
}

// InitHeight: init height from local block
func (n *Node) InitHeight() {
	blkNum, _ := blockchain.GetBlockHeight("../BCData/" + n.NodeID.ID.Name)
	if blkNum > 0 {
		n.BlkStore.Height = int(blkNum)
	}
}

// GetNodeNames: get node names from NodesChannel
func (n *Node) GetNodeNames() []string {
	return n.NodeManager.GetNodeNames()
}

// GetNodeNames: get node except itself names from NodesChannel
func (n *Node) GetOtherNodeNames() []string {
	return n.NodeManager.GetOtherNodeNames()
}

// GetLeader: the node get the leader name of this view
func (n *Node) GetLeader() string {
	return "r_" + strconv.Itoa(n.GetLeaderNum())
}

// Execute: execute the proposal commands
func (n *Node) Execute() bool {
	// for _, v := range n.Consensus.CurProposal.Command {
	// 	fmt.Println(string(v))
	// }
	return true
}

func (n *Node) StartBCNodeJoin(simulateNodes []*Node) {
	n.BasicHotstuff.Logger.Println("StartNodeJoin", len(n.NodeManager.NodesTable))

	n.NodeManager.Mode = 1
	for _, node := range simulateNodes {
		node.NodeManager.NewNode.Chan = n.NodeManager.NewNode.Chan
	}
	// fmt.Println(n.NodeManager.NodesTable)
	for _, nodeKey := range n.NodeManager.NodesTable {
		// fmt.Println(nodeKey.Name)
		if nodeKey.Name == n.NodeID.ID.Name {
			continue
		}
		// p := hstypes.Proposal{
		// 	PreBlkHash: n.NodeID.ID.PubKey,
		// 	RootHash:   nodeKey.Sm4Key,
		// }

		// joinMsg := hstypes.Msg{
		// 	MType:    hstypes.JOIN,
		// 	NMType:   hstypes.JOIN,
		// 	Proposal: p,
		// 	SendNode: n.NodeID.ID.Name,
		// 	ReciNode: nodeKey.Name,
		// }

		nKey := mgmt.NodeKey{
			Name:      n.NodeManager.NewNode.Name,
			Sm2PubKey: n.NodeID.ID.PubKey,
			Sm4Key:    nodeKey.Sm4Key,
		}
		newJoinMsg := mgmt.NodeMgmtMsg{
			Type:     mgmt.JOIN,
			NMType:   mgmt.NM_APPLY,
			NodeKey:  nKey,
			SendNode: n.NodeID.ID.Name,
			ReciNode: nodeKey.Name,
		}

		go n.SendNMMsg(&newJoinMsg)

		// go n.SendBMsg(&joinMsg)
	}
	n.NodeManager.State = mgmt.NM_APPLY
}

func (n *Node) StartBCNodeExit() {
	n.BasicHotstuff.Logger.Println("StartNodeExit", n.NodeID.ID.Name)
	n.NodeManager.Mode = 2
	for _, nodeKey := range n.NodeManager.NodesTable {
		// fmt.Println(nodeKey.Name)
		if nodeKey.Name == n.NodeID.ID.Name {
			n.Stop()
			continue
		}
		// generate the exit message
		exitMsg := mgmt.NodeMgmtMsg{
			Type:     mgmt.EXIT,
			NMType:   mgmt.NM_APPLY,
			Leader:   int(time.Now().UnixNano()) / int(time.Millisecond),
			SendNode: n.NodeID.ID.Name,
			ReciNode: nodeKey.Name,
		}

		// sign, _ := sm2.Sign(rand.Reader, n.NodeID.PrivateKey, exitMsg.Message2Byte())
		go n.SendNMMsg(&exitMsg)
	}
	n.NodeManager.State = mgmt.NM_APPLY
}

func (n *Node) UpdateNodeInfo(index int) {
	if n.NodeID.ID.Name != n.NodeManager.NewNode.Name {
		// the original nodes update node-manager and consensus information
		n.NodeManager.UpdateNewNodeInfo()
		n.BasicHotstuff.UpdateNodesNum(len(n.NodeManager.NodesTable))

		// reset the node-manager state
		n.NodeManager.ResetNodeManager()
	} else {
		// the new node update itself
		n.BasicHotstuff.UpdateNodesNum(len(n.NodeManager.NodesTable))
		n.BasicHotstuff.SyncInfo(n.NodeManager.SyncMsgs[index], n.NodeManager.GetLeaderFromSyncMsgs(n.NodeManager.SyncMsgs))
		n.NodeManager.ResetNodeManager()
	}
}

func (n *Node) DeleteNodeInfo(msg *hstypes.Msg) {
	if n.NodeID.ID.Name != n.NodeManager.NewNode.Name {
		// the original nodes update node-manager and consensus information
		n.NodeManager.UpdateNewNodeInfo()
		n.BasicHotstuff.UpdateNodesNum(len(n.NodeManager.NodesTable))

		// reset the node-manager state
		n.NodeManager.ResetNodeManager()
	}
}

// Sign: sign byte slice message using SM2
func (n *Node) Sign(msg []byte) ([]byte, error) {
	sign, err := sm2.Sm2Sign(n.NodeID.PrivateKey, n.NodeID.ID.PubKey, msg)
	return sign, err
}

// VerifySign: verify signature using SM2
func (n *Node) VerifySign(msg []byte, pubKey []byte, sign []byte) bool {
	return sm2.Sm2Verify(sign, pubKey, msg)
}
