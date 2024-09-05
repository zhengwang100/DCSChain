package factory_test

import (
	common "common"
	"factory"
	"fmt"
	"ssm2"
	"testing"
	"tss"
)

// TestSigners: test whether the Signer generated by function GenSigners works correctly
func TestSigners(t *testing.T) {
	newSigners := factory.GenSigners(common.HOTSTUFF_2_PROTOCOL, 4)
	fmt.Printf("%T", newSigners)
	fmt.Println()
	for i := range newSigners {
		fmt.Printf("%T", newSigners[i])
		fmt.Println()
		tssSigner, ok := newSigners[i].(*tss.Signer)
		if !ok {
			fmt.Println("Signer type does not match!", ok, tssSigner)
			return
		} else {

			fmt.Println("tsss")
		}
		sm2Signer, ok := newSigners[i].(*ssm2.Signer)
		if !ok {
			fmt.Println("Signer type does not match!", ok, sm2Signer)
			return
		} else {
			fmt.Println("ssm2")
		}
	}
}
