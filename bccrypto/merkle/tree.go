package merkle

import (
	"math/bits"
)

// HashFromByteSlices: compute a Merkle tree where the leaves are the byte slice, in the provided order
// params:
// -input: all leaf message
// return root hash of merkle tree
func HashFromByteSlices(input [][]byte) []byte {
	// fmt.Println(len(input))
	switch len(input) {
	case 0:
		return EmptyHash()
	case 1:
		return leafHash(input[0])
	default:
		k := getSplitPoint(int64(len(input)))
		left := HashFromByteSlices(input[:k])
		right := HashFromByteSlices(input[k:])
		return innerHash(left, right)
	}
}

// HashFromByteSlicesIterative: compute a Merkle tree where the leaves are the byte slice by the iterative
// params:
// -input: all leaf message
// return root hash of merkle tree
func HashFromByteSlicesIterative(input [][]byte) []byte {
	items := make([][]byte, len(input))
	// fmt.Println(len(items), cap(input))
	for i, leaf := range input {
		// fmt.Println(i, leaf)
		// items[i] = Sum(leaf)
		items[i] = Sum(append([]byte{0}, leaf...))
		// items[i] = leafHash(leaf)
	}

	size := len(items)
	for {
		switch size {
		case 0:
			return EmptyHash()
		case 1:
			return items[0]
		default:
			rp := 0 // read position
			wp := 0 // write position
			for rp < size {
				if rp+1 < size {
					items[wp] = innerHash(items[rp], items[rp+1])
					rp += 2
				} else {
					items[wp] = items[rp]
					rp++
				}
				wp++
			}
			size = wp
		}
	}
}

// getSplitPoint: get the largest power of 2 less than length
// if the split point equal the length ,that is, the length is an integer multiple of 2, then shift one bit right
// params:
// -length: the message length need to split
// return the split point
func getSplitPoint(length int64) int64 {
	if length < 1 {
		panic("Trying to split a tree with size < 1")
	}
	uLength := uint(length)
	bitlen := bits.Len(uLength)
	k := int64(1 << uint(bitlen-1))
	if k == length {
		k >>= 1
	}
	return k
}
