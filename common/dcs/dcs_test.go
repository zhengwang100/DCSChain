package dcs_test

import (
	"deltachain/common/dcs"
	"fmt"
	"testing"
)

func TestDCS(t *testing.T) {
	n := 4
	th := 300.0
	l := 0.131

	fmt.Println(dcs.GetDCS(n, l, th))
	fmt.Println(dcs.GetDecentralization(n))
	fmt.Println(dcs.GetConsistency(l))
	fmt.Println(dcs.GetScalability(th))
}
