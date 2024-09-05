package hotstuff2cmd

import (
	"blockchain"
	"bufio"
	"ofactory"

	mysm4 "bccrypto/encrypt_sm4"
	"fmt"
	hs2types "hotstuff2/types"
	"log"
	"mgmt"

	common "common"
	"os"
	"strconv"
	"strings"
	"tss"
)

// StartHotstuff2: the main process of chained hotstuff, the current implementation commands are as follows:
// 'r': generate a new request and send to the leader
// 'a': auto generate new requests, three parameters are respectively defined as count, reqNum, length (See function AutoGenChainedNewReq for details)
// 'c': check the chained node informaion
// 'e': exit
func StartHotstuff2(nodeNum int) {

	// define a log object to facilitate log printing
	mainLogger := *log.New(os.Stdout, "", 0)
	mainLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	mainLogger.Println("Hotstuff-2 running")

	// firstly generate new nodes and start the first chained round with command "Genesis block"
	simulateNodes := GenH2Nodes(nodeNum)
	GenHotstuff2FirstRound(simulateNodes)

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
		case "r":
			GenNewH2Req(simulateNodes, input[1:])
		case "a":
			AutoGenH2NewReq(simulateNodes, input[1:])
		case "c":
			CheckH2NodeInfo(simulateNodes)
		case "b":
			CheckH2BlkInfo(simulateNodes)
		case "e":
			break outerLoop
		default:
			mainLogger.Println("Input invalid")
		}
	}
}

// GenH2Nodes: generate the specified number of new chained nodes
func GenH2Nodes(nodeNum int) []*ofactory.Node {

	// declare simulate chained nodes and channel belong to node
	var simulateNodes []*ofactory.Node
	nodesChannel := make(map[string]chan []byte)

	// generate n signers that satisfy the condition of threshold 2f+1
	newSigners := tss.NewSigners(nodeNum, (nodeNum-1)/3*2+1)

	for i := 0; i < nodeNum; i++ {
		nodeName := "r_" + strconv.Itoa(i)
		newNode, err := ofactory.NewNode(i, common.HOTSTUFF_2_PROTOCOL, "./BCData")
		if err != nil {
			fmt.Println("error:", err)
		} else {
			simulateNodes = append(simulateNodes, newNode)
			nodesChannel[nodeName] = make(chan []byte, 64)
		}
		simulateNodes[i].NodeManager.NodesChannel = nodesChannel
		simulateNodes[i].Hotstuff2.ThresholdSigner = newSigners[i]
		simulateNodes[i].Hotstuff2.ConsId = i
		simulateNodes[i].Hotstuff2.ForwardChan = nodesChannel[nodeName]
		simulateNodes[i].Hotstuff2.BlkStore.Path = "./BCData/" + nodeName
		simulateNodes[i].Hotstuff2.View.NodesNum = nodeNum

		// start two process to handle the chained message and handle chained requests
		go simulateNodes[i].H2HandleMsg(simulateNodes[i].NodeManager.NodesChannel[nodeName])
		go simulateNodes[i].H2HandleReq()
	}

	proposalQC := GenEmptyQC(simulateNodes, -1, hs2types.PROPOSE)
	prepareQC := GenEmptyQC(simulateNodes, -1, hs2types.PREPARE)
	// update nodes' route table that record other node information
	for i := 0; i < nodeNum; i++ {
		simulateNodes[i].Hotstuff2.ProposalQC = proposalQC
		simulateNodes[i].Hotstuff2.PrepareQC = prepareQC
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
func GenHotstuff2FirstRound(simulateNodes []*ofactory.Node) {
	dirPath := "./BCData/"
	// msgs := make([][]byte, len(simulateNodes))
	// generate empty proposal to init
	// proposal := hs2types.Proposal{}
	// emptyHash := proposal.GenProposalHash()

	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].NodeID.ID.Name + "/")
	}
	// update the first leader's request
	simulateNodes[0].Hotstuff2.InitLeader()
	simulateNodes[0].Requests = [][]byte{[]byte("Genesis block")}

}

// CheckNodeInfo: print the node information
func CheckH2NodeInfo(nodes []*ofactory.Node) {
	for i := range nodes {
		fmt.Println(nodes[i].NodeID.ID.Name, ":", nodes[i])
	}
}

// GenEmptyQC: generate a valid empyt QC by the qcType
func GenEmptyQC(simulateNodes []*ofactory.Node, viewNumber int, qcType hs2types.StateType) hs2types.QuromCert {

	proposal := hs2types.Proposal{}
	emptyHash := proposal.GenProposalHash()

	emptyMsg := hs2types.H2Msg{
		MType:      qcType,
		ViewNumber: viewNumber,
		Hs2Node: common.HsNode{
			ParentHash: emptyHash,
			CurHash:    emptyHash,
		},
	}

	partSign := make([][]byte, len(simulateNodes))
	for i := 0; i < len(simulateNodes); i++ {
		partSign[i], _ = simulateNodes[i].Hotstuff2.ThresholdSigner.ThresholdSign(emptyMsg.Message2Byte())
	}

	emptyMsgSign, _ := simulateNodes[1].Hotstuff2.ThresholdSigner.CombineSig(emptyMsg.Message2Byte(), partSign[:3])

	emptyQC := hs2types.QuromCert{
		QType:      qcType,
		ViewNumber: emptyMsg.ViewNumber,
		Hs2Node:    emptyMsg.Hs2Node,
		Sign:       emptyMsgSign,
	}

	return emptyQC
}

// GenNewH2Req: generate new request
func GenNewH2Req(simulateNodes []*ofactory.Node, msg []string) {
	for i, v := range simulateNodes {
		if v.Hotstuff2.IsLeader() {
			if v.Hotstuff2.CurPhase == hs2types.NEW_VIEW || v.Hotstuff2.CurPhase == hs2types.NEW_PROPOSE {
				simulateNodes[i].Requests = append(simulateNodes[i].Requests, common.String2ByteSlice(msg)...)
				break
			} else {
				simulateNodes[(i+1)%len(simulateNodes)].Requests = append(simulateNodes[(i+1)%len(simulateNodes)].Requests, common.String2ByteSlice(msg)...)
				break
			}
		}
	}
}

// AutoGenH2NewReq: in basic-hotstuff, auto-generate requests
// params:
// - params: slice of parameters
// -- 1.count: Number of requests
// -- 2.reqNum: The number of transactions per request
// -- 3.length: Length of each request
func AutoGenH2NewReq(simulateNodes []*ofactory.Node, params []string) {
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
		GenNewH2Req(simulateNodes, common.GenerateSecureRandomStringSlice(paramInt[1], paramInt[2]))
		// time.Sleep(time.Millisecond * 2)
	}
}

// CheckBlkInfo: check block info about hash
func CheckH2BlkInfo(simulateNodes []*ofactory.Node) {
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
