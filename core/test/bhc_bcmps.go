package test

/*
bhc_bcmps.go: basic hotstuff consensus with basic management
*/

import (
	mysm4 "bccrypto/encrypt_sm4"
	"bufio"
	common "common"
	"factory"
	"fmt"
	"log"
	"mgmt"
	"os"
	"server"
	"strconv"
	"strings"
	"time"
	"tss"
)

// StartHotstuff: the main process of basic hotstuff, the current implementation commands are as follows:
// 'r': generate a new request and send to the leader
// 'a': auto generate new requests, three parameters are respectively defined as count, reqNum, length (See function AutoGenChainedNewReq for details)
// 'b': check the block information
// 'c': check the chained node information
// 'j': start a new node join the system
// 'e': start a orignal node exit the system
// 'q': exit
func Start(nodeNum int, path string, consType common.ConsensusType, nmType mgmt.NodeManagerType) {

	// define a log object to facilitate log printing
	mainLogger := *log.New(os.Stdout, "", 0)
	mainLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	mainLogger.Println("Server running")

	// firstly generate new nodes and start the first chained round with command "Genesis block"
	simulateServers := factory.GenServers(nodeNum, path, consType, nmType)
	// mainLogger.Println(simulateServers)
	factory.GenFirstRound(simulateServers, path)
	// constantly loop to get commands
outerLoop:
	for {
		reader := bufio.NewReader(os.Stdin)
		// read a line command
		inp, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Err reading:", err)
		}
		inp = strings.TrimRight(inp, "\r\n")
		input := strings.Split(inp, " ")

		// the input command calls the function to process
		switch input[0] {
		case "r":
			factory.GenNewReq(simulateServers, factory.SignCmd([][]byte{[]byte(input[1])}))
		case "a":
			factory.AutoGenNewReq(simulateServers, input[1:])
		case "c":
			fmt.Println(len(simulateServers[1].NodeManager.NodesTable))
			// CheckNodeInfo(simulateServers)
		case "b":
			fmt.Println(factory.CheckBlkInfo(simulateServers))
		case "j":
			NewServerJoin(&simulateServers)
		case "e":
			OriServerExit(&simulateServers, input[1:])
		case "q":
			factory.StopAll(simulateServers)
			break outerLoop
		default:
			mainLogger.Println("Input invalid")
		}
	}
}

// NewServerJoin: create a new node and have it initiate a join request to the system
// params:
// simulateServers: the slice of nodes in system
func NewServerJoin(simulateServers *[]*server.Server) {

	// create new node
	nodeName := "r_" + strconv.Itoa(len(*simulateServers))
	tSigner := factory.GenSigners((*simulateServers)[0].Orderer.ConsType, 1)
	newServer, err := server.NewServer(
		len(*simulateServers),
		len(*simulateServers),
		"./BCData",
		(*simulateServers)[0].Orderer.ConsType,
		mgmt.BASIC,
		tSigner[0],
		make(map[string]chan []byte),
	)
	if err != nil {
		return
	}

	// in the pbft consensus, the new node synchronize the public keys of nodes in the original system
	if (*simulateServers)[0].Orderer.ConsType == common.PBFT {
		for i := 0; i < len(*simulateServers); i++ {
			newServer.Orderer.PBFTConsensus.Signer.Pks[(*simulateServers)[i].ServerID.ID.Name] = (*simulateServers)[i].Orderer.PBFTConsensus.Signer.Pk
			(*simulateServers)[i].Orderer.PBFTConsensus.Signer.Pks[newServer.ServerID.ID.Name] = newServer.Orderer.PBFTConsensus.Signer.Pk
		}
	}

	// update new node information
	newServer.NodeManager.NodesChannel[nodeName] = newServer.ServerID.Address
	// newServer.Orderer.BasicHotstuff.ForwardChan = newServer.NodeManager.NodesChannel[nodeName]

	newServer.NodeManager.NewNode.Name = newServer.ServerID.ID.Name
	newServer.NodeManager.NewNode.NodeKey.Sm2PubKey = newServer.ServerID.ID.PubKey
	newServer.NodeManager.NewNode.Chan = newServer.ServerID.Address

	// add PubKey of nodes in system
	for _, node := range *simulateServers {
		sm4PK := mysm4.GenerateKey()
		newServer.NodeManager.NodesChannel[node.ServerID.ID.Name] = node.ServerID.Address
		newServer.NodeManager.NodesTable[node.ServerID.ID.Name] = mgmt.NodeKey{
			Name:      node.ServerID.ID.Name,
			Sm2PubKey: node.ServerID.ID.PubKey,
			Sm4Key:    sm4PK,
		}
	}

	// start two process to handle the message and handle requests
	go newServer.RouteServerMsg(newServer.NodeManager.NodesChannel[nodeName])
	go newServer.HandleReq()

	go newServer.StartNodeJoin(*simulateServers)

	// add new node to the slice of simulated nodes
	*simulateServers = append(*simulateServers, newServer)
	// fmt.Println("In", len(newServer.NodeManager.NodesTable))
	go func(t int) {
		time.Sleep(time.Duration(t) * time.Millisecond)
		UpdateSigners(*simulateServers)
	}(100)
}

// OriServerExit: the orignal node in system exit from the system
// params:
// simulateServers: the slice of nodes in system
// id: 				the id of exit node
func OriServerExit(simulateNodes *[]*server.Server, id []string) {

	// check the params
	if len(id) == 0 {
		fmt.Println("None param")
		return
	}
	// get the exit node name
	name := "r_" + id[0]
	for index, s := range *simulateNodes {
		if s.ServerID.ID.Name == name {

			// the exit node start the exit program
			s.StartNodeExit()

			// delete the exit node
			*simulateNodes = append((*simulateNodes)[:index], (*simulateNodes)[index+1:]...)
			break
		}
	}
	fmt.Println("In")

	// after set time, update the signers
	go func(t int) {
		time.Sleep(time.Duration(t) * time.Millisecond)
		UpdateSigners(*simulateNodes)
	}(100)
}

// UpdateSigners: update simulateServers' orderer signer
// params:
// simulateServers: the slice of nodes in system
func UpdateSigners(simulateServers []*server.Server) {
	nodeNum := len(simulateServers)
	newsigners := tss.NewSigners(nodeNum, (nodeNum-1)/3*2+1)

	switch simulateServers[0].Orderer.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		for i := 0; i < nodeNum; i++ {
			simulateServers[i].Orderer.BasicHotstuff.ThresholdSigner = newsigners[i]
			simulateServers[i].Orderer.BasicHotstuff.Logger.Println("[SIGNER_UPDATE]:", (simulateServers)[i].ServerID.ID.Name, "succeed")
		}
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		for i := 0; i < nodeNum; i++ {
			simulateServers[i].Orderer.ChainedHotstuff.ThresholdSigner = newsigners[i]
			simulateServers[i].Orderer.ChainedHotstuff.Logger.Println("[SIGNER_UPDATE]:", (simulateServers)[i].ServerID.ID.Name, "succeed")
		}
	case common.HOTSTUFF_2_PROTOCOL:
		for i := 0; i < nodeNum; i++ {
			simulateServers[i].Orderer.Hotstuff2.ThresholdSigner = newsigners[i]
			simulateServers[i].Orderer.Hotstuff2.Logger.Println("[SIGNER_UPDATE]:", (simulateServers)[i].ServerID.ID.Name, "succeed")
		}
	case common.PBFT:
		// for i := 0; i < nodeNum; i++ {
		// 	// simulateServers[i].Orderer.PBFTConsensus.ThresholdSigner = newsigners[i]
		// 	simulateServers[i].Orderer.PBFTConsensus.Logger.Println("[SIGNER_UPDATE]:", (simulateServers)[i].ServerID.ID.Name, "succeed")
		// }
	default:

	}
}
