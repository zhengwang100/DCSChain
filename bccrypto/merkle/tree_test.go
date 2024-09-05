package merkle_test

import (
	"common"
	"fmt"
	"merkle"
	"reflect"
	"testing"
	"time"
)

// TestTree: test the right to genetate merkle tree
func TestTree(t *testing.T) {
	testStr1 := "remove Tom"
	testStr2 := "update Alice Math 90"
	testSlice1 := [][]byte{[]byte(testStr1), []byte(testStr2)}
	h1_com := merkle.HashFromByteSlices(testSlice1)
	h1_iterative := merkle.HashFromByteSlicesIterative(testSlice1)
	fmt.Println(reflect.TypeOf(h1_com))
	fmt.Println(h1_com)
	fmt.Println(h1_iterative)

	testSlice2 := [][]byte{[]byte(testStr2), []byte(testStr1)}
	h2_com := merkle.HashFromByteSlices(testSlice2)
	fmt.Println(reflect.TypeOf(h2_com))
	fmt.Println(h2_com)

	testSlice3 := [][]byte{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {1, 4, 7}}
	h3_com := merkle.HashFromByteSlices(testSlice3)
	h3_iterative := merkle.HashFromByteSlicesIterative(testSlice3)

	fmt.Println(reflect.TypeOf(h3_com))
	fmt.Println(h3_com)
	fmt.Println(h3_iterative)

	testEmptySlice := make([][]byte, 13)
	h4 := merkle.HashFromByteSlicesIterative(testEmptySlice)

	fmt.Println("h4", h4)

}

func TestCompare2Func(t *testing.T) {
	count := 10000
	start1 := time.Now()
	for i := 0; i < count; i++ {
		ranSlice := common.GenerateSecureRandom2ByteSlice(128, 128)
		merkle.HashFromByteSlices(ranSlice)
	}
	fmt.Println(time.Since(start1) / time.Duration(count))

	start2 := time.Now()
	for i := 0; i < count; i++ {
		ranSlice := common.GenerateSecureRandom2ByteSlice(128, 128)
		merkle.HashFromByteSlicesIterative(ranSlice)
	}
	fmt.Println(time.Since(start2) / time.Duration(count))
}
