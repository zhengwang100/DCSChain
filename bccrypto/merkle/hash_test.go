package merkle_test

import (
	common "common"
	"fmt"
	"merkle"
	"ssm2"
	"testing"
	"time"
)

// TestEmptyHash: test the func merkle.EmptyHash
func TestEmptyHash(t *testing.T) {
	fmt.Println(merkle.EmptyHash())
}

func TestHash(t *testing.T) {
	count := 100000
	for i := 0; i < count; i++ {
		inp := common.GenerateSecureRandomByteSlice(128)
		merkle.LeafHash(inp)
		// fmt.Println(hash)
	}
}

// TestSignAndVerify: test time spent signatures and validation
func TestSignAndVerify(t *testing.T) {
	count := 10000
	num := 20
	signer := ssm2.NewSigners(1)[0]
	start := time.Now().UnixNano() / 1e6
outloop:
	for i := 0; i < count; i++ {
		inp := common.StringSlice2TwoDimByteSlice(common.GenerateSecureRandomStringSlice(num, 100))
		sign := make([][]byte, num)
		for i := 0; i < num; i++ {
			sign[i] = signer.Sign(inp[i])
		}
		// hash := merkle.HashFromByteSlices(inp)

		for i := 0; i < num; i++ {
			if !signer.VerifySign("r_0", sign[i], inp[i]) {
				break outloop
			}
			// fmt.Println(signer.VerifySign("r_0", sign[i], inp[i]))
		}
		// fmt.Println(hash)
	}
	end := time.Now().UnixNano() / 1e6
	fmt.Printf("time 		:	%d ms\n", end-start)
	fmt.Printf("num  		:	%d\n", count*num)
	fmt.Printf("ave time  	:	%f ms\n", float64(end-start)/float64(count*num))
}
