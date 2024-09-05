package merkle

import (
	"hash"

	"github.com/xlcetc/cryptogm/sm/sm3"
)

var (
	leafPrefix = []byte{0}
	// leafPrefix = make([]byte, 0, 2^14)
	// leafPrefix  = append(leafPrefix, byte(0))
	innerPrefix = []byte{1}
)

// New: return a new sm3 hash
func New() hash.Hash {
	return sm3.New()
}

// Sum: get hash slices the sm3 of the content
func Sum(content []byte) []byte {
	h := sm3.SumSM3(content)
	return h[:]
}

// EmptyHash: get a hash of empty message
func EmptyHash() []byte {
	return Sum([]byte{})
}

// leafHash: get the hash of (0x00 || leaf)
func leafHash(leaf []byte) []byte {
	// if len(leaf) != 128 {
	// 	fmt.Print(leaf)
	// }
	// fmt.Println(len(leaf) + len(leafPrefix))
	// t := make([]byte, len(leaf)+len(leafPrefix))
	// copy(t, leafPrefix)
	// copy(t[len(leafPrefix):], leaf)
	// return EmptyHash()
	return Sum(leaf)
}

func LeafHash(leaf []byte) []byte {
	// fmt.Println(len(leafPrefix))
	return Sum(append(leafPrefix, leaf...))
}

// innerHash: get the hash of 0x01 || left || right
func innerHash(left []byte, right []byte) []byte {
	return Sum(append(innerPrefix, append(left, right...)...))
}
