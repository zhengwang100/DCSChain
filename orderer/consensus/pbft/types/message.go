package ptypes

import (
	"blockchain"
)

// PMsg: PBFT message
type PMsg struct {
	MType      StateType // this message type
	ViewNumber int       // the view number when the message is sent
	SeqNum     int       // the sequence of transaction corresponding to the message

	Digest    []byte // summary of message
	Signature []byte // signature of message

	Proposal Proposal         // the new proposal
	Block    blockchain.Block // the proposed block in the view

	SendNode string // the sending node of the message
	ReciNode string // the receiving node of the message

	CSet []*PMsg `json:"CSet,omitempty"` // the collection of checkpoint messages
	PSet []*Pm   `json:"PSet,omitempty"` // the collection of Pm
	VSet []*PMsg `json:"VSet,omitempty"` // the collection of the valid view-change messages received by the primary
	OSet []*PMsg `json:"OSet,omitempty"` // the collection of newly generated pre-prepare messages
}

// Pm:  pre-prepare messages and a collection of prepare messages
// Pm implement PBFT description as follow:
// Pm contains a valid pre-prepare message (without the corresponding client message) and 2f matching, valid
// prepare messages signed by different backups with the same view, sequence number, and the digest of m .
type Pm struct {
	PrePrepareMsg *VCMsg   `json:"PrePrepareMsg,omitempty"` // the pre-prepare messages
	PrepareMsgs   []*VCMsg `json:"PrepareMsgs,omitempty"`   // a collection of prepare messages corresponds to the pre-prepare message
}

type VCMsg struct {
	MType      StateType // this message type
	ViewNumber int       // the view number when the message is sent
	SeqNum     int       // the sequence of transaction corresponding to the message

	Digest    []byte // summary of message
	Signature []byte // signature of message

	Proposal Proposal // the new proposal

	SendNode string // the sending node of the message
	ReciNode string // the receiving node of the message
}

// VCMsg2Byte: convert the VCMsg to byte slice
// params:
// - transType: conversion type
// -- 0: convert message type, view number, sequence, digest
// -- 1: convert message type, view number, sequence, digest and sendNode
// return:
// - converted byte slice
func (vcm *VCMsg) VCMsg2Byte(transType int) []byte {
	switch transType {
	case 0:
		return append([]byte{byte(vcm.MType), byte(vcm.ViewNumber), byte(vcm.SeqNum)}, vcm.Digest...)
	case 1:
		res := append([]byte{byte(vcm.MType), byte(vcm.ViewNumber), byte(vcm.SeqNum)}, vcm.Digest...)
		return append(res, []byte(vcm.SendNode)...)
	default:
		return nil
	}
}

// Message2Byte: convert message to byte slice
// params:
// - transType: conversion type
// -- 0 : return message type, view number, sequence number, digest
// -- 1 : return message type, view number, sequence number, digest, send node ID
// -- 2 : return message type, view number, sequence number, digest, Cset, PSet; only for view-change message
// -- 3 : return message type, view number, sequence number, digest, VSet, OSet; only for new-view message
// return:
// - converted byte slice
func (m *PMsg) Message2Byte(transType int) []byte {
	switch transType {
	case 0:
		return append([]byte{byte(m.MType), byte(m.ViewNumber), byte(m.SeqNum)}, m.Digest...)
	case 1:
		res := append([]byte{byte(m.MType), byte(m.ViewNumber), byte(m.SeqNum)}, m.Digest...)
		return append(res, []byte(m.SendNode)...)
	case 2:
		res := append([]byte{byte(m.MType), byte(m.ViewNumber), byte(m.SeqNum)}, m.Digest...)
		for i := 0; i < len(m.CSet); i++ {
			res = append(res, m.CSet[i].Message2Byte(1)...)
		}
		for i := 0; i < len(m.PSet); i++ {
			if m.PSet[i] == nil {
				break
			}
			res = append(res, m.PSet[i].Pm2Byte()...)
		}
		return append(res, []byte(m.SendNode)...)
	case 3:
		res := []byte{byte(m.MType), byte(m.ViewNumber)}
		for i := 0; i < len(m.VSet); i++ {
			res = append(res, m.VSet[i].Message2Byte(0)...)
		}
		for i := 0; i < len(m.OSet); i++ {
			res = append(res, m.OSet[i].Message2Byte(0)...)
		}
		return res
	default:
		return []byte{byte(m.MType), byte(m.ViewNumber)}
	}
}

// Pm2Byte: convert Pm to byte slice
func (pm *Pm) Pm2Byte() []byte {
	res := make([]byte, 0)
	res = append(res, pm.PrePrepareMsg.VCMsg2Byte(0)...)
	for i := 0; i < len(pm.PrepareMsgs); i++ {
		res = append(res, pm.PrepareMsgs[i].VCMsg2Byte(1)...)
	}
	return res
}

// Pm2VCMsg: convert Pm to VCMsg
func (m *PMsg) PMsg2VCMsg() *VCMsg {
	return &VCMsg{
		MType:      m.MType,
		ViewNumber: m.ViewNumber,
		SeqNum:     m.SeqNum,
		Digest:     m.Digest,
		Signature:  m.Signature,
		SendNode:   m.SendNode,
		ReciNode:   m.ReciNode,
		Proposal:   m.Proposal,
	}
}
