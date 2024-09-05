package hs2types

import (
	"blockchain"
	"common"
)

// H2Msg: hotstuff-2 message
type H2Msg struct {
	MType      StateType        // this message type
	ViewNumber int              // the view number when the message is sent
	Hs2Node    common.HsNode    // consist of the hash corresponding to current block and with parent hash
	Block      blockchain.Block // the proposed block in the view
	ConsSign   []byte           // the partial or combined signature
	Justify1   QuromCert        // the single certification QC which is corresponds to proposal QC in hotstuff
	Justify2   QuromCert        // the double certification QC which is corresponds to prepare QC in hotstuff

	SendNode string // the sending node of the message
	ReciNode string // the receiving node of the message
}

// Message2Byte: convert message to byte slice
func (m *H2Msg) Message2Byte() []byte {
	return append([]byte{byte(m.MType), byte(m.ViewNumber)}, m.Hs2Node.Object2Byte()...)
}
