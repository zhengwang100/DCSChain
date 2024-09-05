package hs2types

import "common"

// QuromCert: the qurom certification for a block or proposal, QC for short
type QuromCert struct {
	QType      StateType     // the QC type, proposal or prepare in hotstuff-2
	ViewNumber int           // the view number when the QC was generated
	Height     int           // the height of block corresponding to the QC
	Hs2Node    common.HsNode // the hash of block corresponding to the QC and last block
	Sign       []byte        // the combined signature
}

// QC2SignMsgByte: convert QuromCert to byte slice
func (q *QuromCert) QC2SignMsgByte() []byte {
	return append([]byte{byte(q.QType), byte(q.ViewNumber)}, q.Hs2Node.Object2Byte()...)
}
