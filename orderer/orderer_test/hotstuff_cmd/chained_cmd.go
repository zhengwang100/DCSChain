package hotstuffcmd

import (
	mysm4 "bccrypto/encrypt_sm4"
	"blockchain"
	"bufio"
	"fmt"
	hstypes "hotstuff/types"
	"log"
	"mgmt"
	"ofactory"

	common "common"

	"os"
	"strconv"
	"strings"
	"tss"
)

// StartChainedHotstuff: the main process of chained hotstuff, the current implementation commands are as follows:
// 'r': generate a new request and send to the leader
// 'a': auto generate new requests, three parameters are respectively defined as count, reqNum, length (See function AutoGenChainedNewReq for details)
// 'c': check the chained node informaion
// 'e': exit
func StartChainedHotstuff(nodeNum int) {

	// define a log object to facilitate log printing
	mainLogger := *log.New(os.Stdout, "", 0)
	mainLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	mainLogger.Println("ChainedHotstuff running")

	// firstly generate new nodes and start the first chained round with command "Genesis block"
	simulateNodes := GenChainedNodes(nodeNum)
	GenChainedFirstRound(simulateNodes)

	// constantly loop to get commands
outerLoop:
	for {
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
		case "a":
			AutoGenChainedNewReq(simulateNodes, input[1:])
		case "b":
			CheckChainedBlkInfo(simulateNodes)
		case "c":
			CheckNodeInfo(simulateNodes)
		case "e":
			break outerLoop
		case "r":
			GenNewChainedReq(simulateNodes, input[1:])
		default:
			mainLogger.Println("Input invalid")
		}
	}
}

// GenChainedNodes: generate the specified number of new chained nodes
func GenChainedNodes(nodeNum int) []*ofactory.Node {

	// declare simulate chained nodes and channel belong to node
	var simulateNodes []*ofactory.Node
	nodesChannel := make(map[string]chan []byte)

	// generate n signers that satisfy the condition of threshold 2f+1
	newSigners := tss.NewSigners(nodeNum, (nodeNum-1)/3*2+1)

	for i := 0; i < nodeNum; i++ {
		nodeName := "r_" + strconv.Itoa(i)
		newNode, err := ofactory.NewNode(i, common.HOTSTUFF_PROTOCOL_CHAINED, "./BCData")
		if err != nil {
			fmt.Println("error:", err)
		} else {
			simulateNodes = append(simulateNodes, newNode)
			nodesChannel[nodeName] = make(chan []byte, 64)
		}
		simulateNodes[i].NodeManager.NodesChannel = nodesChannel
		simulateNodes[i].ChainedHotstuff.ThresholdSigner = newSigners[i]
		simulateNodes[i].ChainedHotstuff.ConsId = i
		simulateNodes[i].ChainedHotstuff.BlkStore.Path = "./BCData/" + nodeName
		simulateNodes[i].ChainedHotstuff.ForwardChan = nodesChannel[nodeName]
		simulateNodes[i].ChainedHotstuff.View.NodesNum = nodeNum

		// start two process to handle the chained message and handle chained requests
		go simulateNodes[i].CHandleMsg(simulateNodes[i].NodeManager.NodesChannel[nodeName])
		go simulateNodes[i].CHandleReq()
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
func GenChainedFirstRound(simulateNodes []*ofactory.Node) {
	dirPath := "./BCData/"

	// generate empty proposal to init
	// proposal := hstypes.Proposal{}
	// emptyHash := proposal.GenProposalHash()

	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].NodeID.ID.Name + "/")

		// generate the new chained message
		// hsNodes := [4]common.HsNode{{
		// 	CurHash:    emptyHash,
		// 	ParentHash: emptyHash,
		// }}
		// msg := hstypes.CMsg{
		// 	MType:      hstypes.NEW_VIEW,
		// 	ViewNumber: 0,
		// 	SendNode:   "r_" + strconv.Itoa(i),
		// 	ReciNode:   "r_0",
		// 	HsNodes:    hsNodes,
		// 	Justify: hstypes.ChainedQC{
		// 		QType:      hstypes.NEW_VIEW,
		// 		ViewNumber: -1,
		// 		HsNodes:    hsNodes,
		// 		Sign:       nil,
		// 	},
		// 	PartialSig: nil,
		// }

		// // encode the message and send it
		// msgJson, err := json.Marshal(msg)
		// if err == nil {
		// 	(simulateNodes[i]).NodeManager.NodesChannel["r_0"] <- msgJson
		// }
	}

	// update the first leader's request
	simulateNodes[0].ChainedHotstuff.InitLeader()
	simulateNodes[0].Requests = [][]byte{[]byte("Genesis block")}
}

// GenNewReq: generate new request
// note: if the view leader is waiting for request, send the request to the leader, or send it to next view leader
func GenNewChainedReq(simulateNodes []*ofactory.Node, msg []string) {
	for i, v := range simulateNodes {
		if v.ChainedHotstuff.IsLeader() {
			if v.ChainedHotstuff.CurPhase == hstypes.WAITING {
				simulateNodes[i].Requests = append(simulateNodes[i].Requests, common.String2ByteSlice(msg)...)
			} else {
				simulateNodes[(i+1)%len(simulateNodes)].Requests = append(simulateNodes[(i+1)%len(simulateNodes)].Requests, common.String2ByteSlice(msg)...)
			}
		}
	}
}

// AutoGenChainedNewReq: in chained-hotstuff, auto-generate requests
// params
// 1.count: Number of requests
// 2.reqNum: The number of transactions per request
// 3.length: Length of each request
func AutoGenChainedNewReq(simulateNodes []*ofactory.Node, params []string) {

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
		GenNewChainedReq(simulateNodes, common.GenerateSecureRandomStringSlice(paramInt[1], paramInt[2]))
		// time.Sleep(time.Millisecond * 2)
	}
}

// CheckChainedNodeInfo: print the chained node information
func CheckChainedNodeInfo(simulateNodes []*ofactory.Node) {
	for i := range simulateNodes {
		fmt.Println(simulateNodes[i].NodeID.ID.Name, ":", simulateNodes[i])
	}
}

// CheckChainedBlkInfo: check block info about hash
func CheckChainedBlkInfo(simulateNodes []*ofactory.Node) {
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
