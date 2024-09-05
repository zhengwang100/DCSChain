package main

import (
	hotstuff2cmd "deltachain/orderer/orderer_test/hotstuff_2_cmd"
	hotstuffcmd "deltachain/orderer/orderer_test/hotstuff_cmd"
	pbftcmd "deltachain/orderer/orderer_test/pbft_cmd"
	"flag"
	"fmt"
	"os"
)

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
		hotstuffcmd.StartBasicHotstuff(node, path)
	case "ch":
		hotstuffcmd.StartChainedHotstuff(node)
	case "h2":
		hotstuff2cmd.StartHotstuff2(node)
	case "pbft":
		pbftcmd.StartPBFT(node)
	default:
		fmt.Println("Input invalid")
	}
}
