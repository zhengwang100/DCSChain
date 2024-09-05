package mgmt

import (
	"blockchain"
	"common"
	hstypes "hotstuff/types"
	hs2types "hotstuff2/types"
)

// node manager type
type NodeManagerType string

const (
	BASIC         NodeManagerType = "basic"
	BFT_SMART     NodeManagerType = "bftsmart"
	BASED_HISTORY NodeManagerType = "basedhistory"
)

// NodeInfo: a node information applying to join or exit
// NodeInfo contents:
// Sm2Pubkey: used to encrypt and decrypt messages
type NodeInfo struct {
	Name    string
	NodeKey NodeKey
	Chan    chan []byte `json:"NodeInfoChann"` // address in p2p
}

type NodeKey struct {
	Name      string
	Sm2PubKey []byte
	Sm4Key    []byte
}

type StateType uint8

const (
	NM_INACTIVE StateType = iota
	NM_APPLY
	NM_PREPARE_VOTE
	NM_PREPARE_CERT
	NM_COMMIT_VOTE
	NM_COMMIT_CERT
	NM_SYNC
	NM_SYNC_FLAG
	NM_AGREE
	NM_RESTART
)

func (st StateType) String() string {
	switch st {
	case 0:
		return "NM_INACTIVE"
	case 1:
		return "NM_APPLY"
	case 2:
		return "NM_PREPARE_VOTE"
	case 3:
		return "NM_PREPARE_CERT"
	case 4:
		return "NM_COMMIT_VOTE"
	case 5:
		return "NM_COMMIT_CERT"
	case 6:
		return "NM_SYNC"
	case 7:
		return "NM_SYNC_FLAG"
	case 8:
		return "NM_AGREE"
	case 9:
		return "NM_RESTART"
	default:
		return ""
	}
}

// NodeManagerMode: the mode indicates whether a node wants to join or exit
type NodeManagerMode uint8

const (
	NONE NodeManagerMode = iota
	JOIN
	EXIT
)

type NodeMgmtMsg struct {
	Type       NodeManagerMode // this message type
	NMType     StateType       // this message type in join or exit phase
	ViewNumber int             // the view number while generating the message
	Leader     int
	HsNodes    []common.HsNode // hotstuff node
	Justify    hstypes.QC      // qurom certificate
	H2Justify  hs2types.QuromCert
	CJustify   hstypes.ChainedQC
	// Justify    interface{}      // qurom certificate
	NodeKey  NodeKey
	Sign     []byte             // signature
	Block    []blockchain.Block // the proposed block in the view
	SendNode string             // the message send node
	ReciNode string             // the message recieve node
	// Proposal   Proposal            // the new proposal
}
