package hotstuffcmd_test

import (
	hotstuffcmd "deltachain/orderer/orderer_test/hotstuff_cmd"
	"fmt"
	"testing"
	"time"
)

// TestChain: test the chained-hotstuff node
func TestChain(t *testing.T) {
	simulateNodes := hotstuffcmd.GenChainedNodes(4)
	hotstuffcmd.GenChainedFirstRound(simulateNodes)

	count := "10000"
	reqNum := "10"  //The number of transactions per request
	length := "100" //Length of each request
	fmt.Println(time.Now())
	hotstuffcmd.AutoGenChainedNewReq(simulateNodes, []string{count, reqNum, length})

	time.Sleep(10 * time.Second)
}

// TestTime: test time intervals
func TestTime(t *testing.T) {
	startTime := time.Now()

	// simulate some time-consuming operations
	time.Sleep(100 * time.Millisecond)

	// calculates the interval from startTime to the current time
	duration := time.Since(startTime)

	fmt.Println(duration)
}
