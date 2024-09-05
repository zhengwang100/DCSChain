package pbftcmd

import (
	mysm4 "bccrypto/encrypt_sm4"
	"blockchain"
	"bufio"
	common "common"
	"fmt"
	"log"
	"mgmt"
	"ofactory"
	"os"
	ptypes "pbft/types"
	ssm2 "ssm2"
	"strconv"
	"strings"
	"time"
)

// StartHotstuff2: the main process of chained hotstuff, the current implementation commands are as follows:
// 'r': generate a new request and send to the leader
// 'a': auto generate new requests, three parameters are respectively defined as count, reqNum, length (See function AutoGenChainedNewReq for details)
// 'c': check the chained node informaion
// 'e': exit
func StartPBFT(NodeNum int) {

	// define a log object to facilitate log printing
	mainLogger := *log.New(os.Stdout, "", 0)
	mainLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	mainLogger.Println("PBFT running")

	// firstly generate new nodes and start the first chained round with command "Genesis block"
	simulateNodes := GenPBFTNodes(NodeNum)
	GenPBFTFirstRound(simulateNodes)

	// constantly loop to get commands
outerLoop:
	for {
		// time.Sleep(200 * time.Millisecond)
		// fmt.Print(">>> ")
		reader := bufio.NewReader(os.Stdin)

		// read a line command
		inp, err := reader.ReadString('\n')
		if err != nil {
			mainLogger.Println("Err reading:", err)
		}
		inp = strings.TrimRight(inp, "\r\n")
		input := strings.Split(inp, " ")

		// the input command calls the function to process
		switch input[0] {
		case "r":
			GenNewPBFTReq(simulateNodes, input[1:])
		case "a":
			AutoGenPBFTNewReq(simulateNodes, input[1:])
		case "c":
			CheckPBFTNodeInfo(simulateNodes)
		case "b":
			CheckPBFTBlkInfo(simulateNodes)
		case "e":
			break outerLoop
		default:
			mainLogger.Println("Input invalid")
		}
	}
}

// GenNewPBFTReq: generate new request
func GenNewPBFTReq(simulateNodes []*ofactory.Node, msg []string) {
	for i, v := range simulateNodes {
		if v.PBFTConsensus.IsLeader() {
			if v.PBFTConsensus.CurPhase == ptypes.NEW_VIEW {
				simulateNodes[i].Requests = append(simulateNodes[i].Requests, common.String2ByteSlice(msg)...)
			} else {
				simulateNodes[(i+1)%len(simulateNodes)].Requests = append(simulateNodes[(i+1)%len(simulateNodes)].Requests, common.String2ByteSlice(msg)...)
			}
			break
		}
	}
}

// GenPBFTNodes: generate the specified number of new chained nodes
func GenPBFTNodes(nodeNum int) []*ofactory.Node {

	// declare simulate chained nodes and channel belong to node
	var simulateNodes []*ofactory.Node
	nodesChannel := make(map[string]chan []byte)
	cPMsgs := make([]*ptypes.PMsg, nodeNum)

	pSigners := ssm2.NewSigners(nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodeName := "r_" + strconv.Itoa(i)
		newNode, err := ofactory.NewNode(i, common.PBFT, "./BCData")
		if err != nil {
			fmt.Println("error:", err)
		} else {
			simulateNodes = append(simulateNodes, newNode)
			nodesChannel[nodeName] = make(chan []byte, 1024)
		}
		simulateNodes[i].NodeManager.NodesChannel = nodesChannel
		simulateNodes[i].PBFTConsensus.ConsId = i
		simulateNodes[i].PBFTConsensus.ForwardChan = nodesChannel[nodeName]
		simulateNodes[i].PBFTConsensus.Signer = pSigners[i]
		simulateNodes[i].PBFTConsensus.BlkStore.Path = "./BCData/" + nodeName
		simulateNodes[i].PBFTConsensus.View.NodesNum = nodeNum
		cPMsgs[i] = &ptypes.PMsg{
			ViewNumber: 0,
			SeqNum:     0,
			SendNode:   nodeName,
		}
		simulateNodes[i].PBFTConsensus.CheckPoint.CPMsgs = cPMsgs

		// start two process to handle the chained message and handle chained requests
		go simulateNodes[i].PHandleMsg(simulateNodes[i].NodeManager.NodesChannel[nodeName])
		go simulateNodes[i].PHandleReq()
	}

	// update nodes' route table that record other node information
	for i := 0; i < nodeNum; i++ {
		for j := 0; j < nodeNum; j++ {
			if j != i {
				simulateNodes[i].NodeManager.NodesTable[simulateNodes[j].NodeID.ID.Name] = mgmt.NodeKey{
					Name:      simulateNodes[j].NodeID.ID.Name,
					Sm2PubKey: simulateNodes[j].NodeID.ID.PubKey,
				}

				if j > i {
					sm4PK := mysm4.GenerateKey()
					simulateNodes[i].NodeManager.UpdateSm4Key(simulateNodes[j].NodeID.ID.Name, sm4PK)
					simulateNodes[j].NodeManager.UpdateSm4Key(simulateNodes[i].NodeID.ID.Name, sm4PK)
				}
			}
		}
	}
	return simulateNodes
}

// GenChainedFirstRound: generate the first request with command 'Genesis block' and start the first consensus round
func GenPBFTFirstRound(simulateNodes []*ofactory.Node) {
	dirPath := "./BCData/"

	// update the first leader's request
	simulateNodes[0].Requests = [][]byte{[]byte("Genesis block")}

	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].NodeID.ID.Name + "/")
	}

	msgReturn := simulateNodes[0].PBFTConsensus.Preprepare()
	if msgReturn != nil {
		simulateNodes[0].SendPMsg(msgReturn)
	}
}

// AutoGenChainedNewReq: in pbft, auto-generate requests
// params:
// - params: slice of parameters
// -- 1.count: Number of requests
// -- 2.reqNum: The number of transactions per request
// -- 3.length: Length of each request
func AutoGenPBFTNewReq(simulateNodes []*ofactory.Node, params []string) {
	// analytic three parameters, default parameters are (1,1,10)
	paramInt := make([]int, 3)
	if len(params) < 3 {
		paramInt[0], paramInt[1], paramInt[2] = 1, 1, 10
	} else {
		for i := range paramInt {
			val, err := strconv.Atoi(params[i])
			if err == nil {
				paramInt[i] = val
			} else {
				paramInt[i] = i
			}
		}
	}

	// generate chained request according to the parameters
	//
	for i := 0; i < paramInt[0]; i++ {
		GenNewPBFTReq(simulateNodes, common.GenerateSecureRandomStringSlice(paramInt[1], paramInt[2]))
		// time.Sleep(time.Millisecond * 2)
		time.Sleep(time.Millisecond * 10)
	}
}

// CheckPBFTNodeInfo: print the node information
func CheckPBFTNodeInfo(simulateNodes []*ofactory.Node) {
	for _, n := range simulateNodes {
		fmt.Println(n.NodeID.ID.Name, n.PBFTConsensus.CurPhase)
	}
}

// CheckPBFTBlkInfo: check block info about hash
func CheckPBFTBlkInfo(simulateNodes []*ofactory.Node) {
	height, err := blockchain.GetBlockHeight("E:/MyOwnDoc/Project/GoProject/src/DeltaChain/BCData/" + simulateNodes[0].NodeID.ID.Name)
	if err != nil {
		fmt.Println(err)
	}
	for i := 0; i <= int(height); i++ {
		blk, _ := simulateNodes[0].BlkStore.ReadBlock("E:/MyOwnDoc/Project/GoProject/src/DeltaChain/BCData/"+simulateNodes[0].NodeID.ID.Name+"/", i)
		fmt.Println("Height "+strconv.Itoa(i)+" pre", blk.BlkHdr.PreBlkHash)
		fmt.Println(blk.Hash())
	}
}
