package hs2types

import (
	common "common"
	"merkle"
)

// Proposal: the proposal which a view of the consensus on, in other words, the proposal and the block are almost equivalent
type Proposal struct {
	Height     int      // the height of this block
	ViewNumber int      // the view number of this proposal or block
	PreBlkHash []byte   // hash of the previous block
	RootHash   []byte   // the root hash of the Merkle tree for the current block
	Command    [][]byte // the requests or commands packaged in this proposal or block
}

// IsEmpty:check whether the proposal is empty
func (p *Proposal) IsEmpty() bool {
	return p.Height == 0 && p.Command == nil
}

// GenProposalHash: generate the proposal hash
// note: if the proposal is empty, return the hash of '[]byte{}'
func (p *Proposal) GenProposalHash() []byte {
	if p.IsEmpty() {
		return merkle.EmptyHash()
	}
	proContent := make([]byte, 0)
	proContent = append(proContent, byte(p.Height))
	proContent = append(proContent, byte(p.ViewNumber))
	proContent = append(proContent, p.PreBlkHash...)
	proContent = append(proContent, p.RootHash...)
	proContent = append(proContent, common.TwoDimByteSlice2OneDimByteSlice(p.Command)...)
	return merkle.Sum(proContent)
}
