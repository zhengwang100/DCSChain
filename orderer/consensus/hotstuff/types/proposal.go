package hstypes

import (
	common "common"
	"merkle"
)

// Proposal: the porposal in hotstuff
type Proposal struct {
	Height     int      // the block height of the proposal
	PreBlkHash []byte   // the hash of previos block
	RootHash   []byte   // the root of merkle tree of commands
	Commands   [][]byte // the node recieved commands
	Signs      [][]byte // the node recieved commands
	Qc         QC       // to ensure security this node attaches a certificate
}

// IsEmpty:check whether the proposal is empty
func (p *Proposal) IsEmpty() bool {
	return p.Commands == nil && p.RootHash == nil && p.PreBlkHash == nil
}

// GenProposalHash: generate the proposal hash
// note: if the proposal is empty, return the hash of '[]byte{}'
func (p *Proposal) GenProposalHash() []byte {
	if p.IsEmpty() {
		return merkle.EmptyHash()
	}
	proContent := make([]byte, 0)
	proContent = append(proContent, byte(p.Height))
	proContent = append(proContent, p.PreBlkHash...)
	proContent = append(proContent, p.RootHash...)
	proContent = append(proContent, common.TwoDimByteSlice2OneDimByteSlice(p.Commands)...)
	return merkle.Sum(proContent)
}
