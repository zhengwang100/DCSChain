package hstypes

import (
	"blockchain"
	common "common"
)

// CMsg: chained message in system
type CMsg struct {
	MType      StateType        // this chained message type
	ViewNumber int              // the view number while generating the message
	SendNode   string           // the chained message send node
	ReciNode   string           // the chained message recieve node
	HsNodes    [4]common.HsNode // four hotstuff nodes
	Justify    ChainedQC        // qurom certificate
	PartialSig []byte           // part signature
	Proposal   Proposal         // the new proposal
	Blk        blockchain.Block // block
}

// ChainedMessage2Byte: convert chained message to byte slice
// return:
// - the byte slice of chined message
// note: the difference is that chained message has four hotstuff node but basic message has only one
func (cm *CMsg) ChainedMessage2Byte() []byte {
	qcByte := []byte{byte(cm.MType), byte(cm.ViewNumber)}
	for i := range cm.HsNodes {
		qcByte = append(qcByte, cm.HsNodes[i].Object2Byte()...)
	}
	return qcByte
}
