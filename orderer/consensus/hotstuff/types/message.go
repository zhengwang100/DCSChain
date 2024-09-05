package hstypes

import (
	"blockchain"
	common "common"
)

// Msg: message in system
type Msg struct {
	MType      StateType     // this message type
	NMType     StateType     // this message type in join or exit phase
	ViewNumber int           // the view number while generating the message
	HsNode     common.HsNode // hotstuff node
	Justify    QC            // qurom certificate
	PartialSig []byte        // part signature
	// Signs      [][]byte
	Proposal Proposal         // the new proposal
	Block    blockchain.Block // the proposed block in the view
	SendNode string           // the message send node
	ReciNode string           // the message recieve node
}

// Message2Byte: convert message to byte slice
func (m *Msg) Message2Byte() []byte {
	return append([]byte{byte(m.MType), byte(m.ViewNumber)}, m.HsNode.Object2Byte()...)
}
