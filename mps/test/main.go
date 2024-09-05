package main

import (
	"common"
	"flag"
	"fmt"
	"mgmt"
	"os"
	"test"
)

// this is started only to test whether the function of the node joining or exiting is complete.
func main() {
	fmt.Println("Welcome to XBC!")
	protocolPtr := flag.String("pr", "", "The protocol to use")
	nodePtr := flag.Int("n", 0, "The node number")
	pathPtr := flag.String("pa", "./BCData", "Storage path for blocks ")
	helpPtr := flag.Bool("h", false, "Display this help message")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	// parse command line arguments
	flag.Parse()

	if *helpPtr {
		flag.Usage()
		return
	}

	// gets the value of a command line argument
	protocol := *protocolPtr
	node := *nodePtr
	path := *pathPtr

	switch protocol {
	case "bh":
		// hotstuff.StartBasicHotstuff(node, path)
		test.Start(node, path, common.HOTSTUFF_PROTOCOL_BASIC, mgmt.BASIC)
	case "ch":
		test.Start(node, path, common.HOTSTUFF_PROTOCOL_CHAINED, mgmt.BASIC)
	case "h2":
		test.Start(node, path, common.HOTSTUFF_2_PROTOCOL, mgmt.BASIC)
	case "pbft":
		test.Start(node, path, common.PBFT, mgmt.BASIC)
	default:
		fmt.Println("Input invalid")
	}
	// }
}
