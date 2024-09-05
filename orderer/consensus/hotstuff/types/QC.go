package hstypes

import common "common"

// QC: qurom certificate
type QC struct {
	QType      StateType     //
	ViewNumber int           // the view number
	HsNode     common.HsNode // the qurom certificate node
	Sign       []byte        // part signature or complete signature
}

// QC2SignMsgByte: convert QC to signed message's byte slice
func (q *QC) QC2SignMsgByte() []byte {
	return append([]byte{byte(q.QType), byte(q.ViewNumber)}, q.HsNode.Object2Byte()...)
}

const MsgBufferLength uint8 = 8
