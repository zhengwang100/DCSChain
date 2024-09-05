package hotstuffcmd_test

import (
	hotstuffcmd "deltachain/orderer/orderer_test/hotstuff_cmd"
	"fmt"
	"testing"
)

// TestGenNodes: test the basic-hotstuff node
func TestGenNodes(t *testing.T) {
	simulateNodes := hotstuffcmd.GenNodes(4, "./BCData")
	fmt.Println(len(simulateNodes))
}
