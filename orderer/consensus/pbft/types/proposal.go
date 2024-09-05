package ptypes

import (
	"merkle"

	common "common"
)

// Proposal: the porposal in PBFT
type Proposal struct {
	Height     int      // the block height of the proposal
	ViewNumber int      // the view number of the proposal
	PreBlkHash []byte   // the hash of previos block
	CurBlkHash []byte   //the hash of current block
	Command    [][]byte // the node recieved commands
	// Qc         QC
}

// IsEmpty: check whether Proposal is empty
func (p Proposal) IsEmpty() bool {
	return p.Height == 0 && p.Command == nil
}

// GetCommandsDigest: get a summary of the order in the proposal
func (p Proposal) GetCommandsDigest() []byte {
	return merkle.Sum(common.TwoDimByteSlice2OneDimByteSlice(p.Command))
}
