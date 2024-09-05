package common

// HsNode: the hotstuff node
type HsNode struct {
	CurHash    []byte // hash of the current block
	ParentHash []byte // node's parent, hash of previous blcok
	// Block  *types.Block // named cmd in paper
}

// Object2Byte: convert hsnode to byte slice, simple link
// return:
// - byte slice of hotstuff node
func (h *HsNode) Object2Byte() []byte {
	return append(h.CurHash, h.ParentHash...)
}
