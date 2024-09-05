package hotstuffcmd

import (
	mysm4 "bccrypto/encrypt_sm4"
	"blockchain"
	"bufio"
	"fmt"
	hstypes "hotstuff/types"
	"log"
	"merkle"
	"mgmt"
	"ofactory"

	common "common"

	"os"
	"strconv"
	"strings"
	"time"
	"tss"
)

// StartHotstuff: the main process of basic hotstuff, the current implementation commands are as follows:
// 'r': generate a new request and send to the leader
// 'a': auto generate new requests, three parameters are respectively defined as count, reqNum, length (See function AutoGenChainedNewReq for details)
// 'c': check the chained node information
// 'e': exit
func StartBasicHotstuff(nodeNum int, path string) {

	// define a log object to facilitate log printing
	mainLogger := *log.New(os.Stdout, "", 0)
	mainLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	mainLogger.Println("BasicHotstuff running")

	// firstly generate new nodes and start the first chained round with command "Genesis block"
	simulateNodes := GenNodes(nodeNum, path)

	GenFirstRound(simulateNodes)
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
			GenNewReq(simulateNodes, input[1:])
		case "a":
			AutoGenNewReq(simulateNodes, input[1:])
		case "e":
			break outerLoop
		case "c":
			fmt.Println(len(simulateNodes[1].NodeManager.NodesTable))
			// CheckNodeInfo(simulateNodes)
		case "b":
			CheckBlkInfo(simulateNodes)
		default:
			mainLogger.Println("Input invalid")
		}
	}
}

// GenNodes: generate the specified number of new nodes
func GenNodes(nodeNum int, path string) []*ofactory.Node {

	// declare simulate nodes and channel belong to node
	var simulateNodes []*ofactory.Node
	nodesChannel := make(map[string]chan []byte)
	emptyMsg := hstypes.Msg{
		ViewNumber: -1,
	}
	// generate n signers that satisfy the condition of threshold 2f+1
	newSigners := tss.NewSigners(nodeNum, (nodeNum-1)/3*2+1)
	for i := 0; i < nodeNum; i++ {
		nodeName := "r_" + strconv.Itoa(i)
		newNode, err := ofactory.NewNode(i, common.HOTSTUFF_PROTOCOL_BASIC, path)
		if err != nil {
			fmt.Println("error:", err)
		} else {
			simulateNodes = append(simulateNodes, newNode)
			nodesChannel[nodeName] = newNode.NodeID.Address
		}
		simulateNodes[i].NodeManager.NodesChannel = nodesChannel
		simulateNodes[i].BasicHotstuff.ThresholdSigner = newSigners[i]
		simulateNodes[i].BasicHotstuff.ConsId = i
		simulateNodes[i].BasicHotstuff.BlkStore.Path = path + "\\" + nodeName
		simulateNodes[i].BasicHotstuff.ForwardChan = newNode.NodeID.Address
		simulateNodes[i].BasicHotstuff.LastRoundMsg = append(simulateNodes[i].BasicHotstuff.LastRoundMsg, &emptyMsg)
		simulateNodes[i].BasicHotstuff.View.NodesNum = nodeNum

		// start two process to handle the message and handle requests
		go simulateNodes[i].HandleMsg(simulateNodes[i].NodeManager.NodesChannel[nodeName])
		go simulateNodes[i].HandleReq()
	}

	// update nodes' route table that record other node information
	for i := 0; i < nodeNum; i++ {
		for j := 0; j < nodeNum; j++ {
			if j != i {
				// simulateNodes[i].NodesTable[simulateNodes[j].NodeID.ID.Name] = simulateNodes[j].NodeID.ID.PubKey
				// fmt.Println(simulateNodes[i].NodeManager.NodesTable)
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

// CheckNodeInfo: print the node information
func CheckNodeInfo(simulateNodes []*ofactory.Node) {
	for i := range simulateNodes {
		fmt.Println(simulateNodes[i].NodeID.ID.Name, ":", simulateNodes[i])
	}
}

// GenChainedFirstRound: generate the first request with command 'Genesis block' and start the first consensus round
func GenFirstRound(simulateNodes []*ofactory.Node) {
	dirPath := "./BCData/"

	// emptyQC := GenEmptyQC(simulateNodes, hstypes.PREPARE)
	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].NodeID.ID.Name + "/")

	}

	// update the first leader's request
	simulateNodes[0].BasicHotstuff.InitLeader()
	simulateNodes[0].Requests = [][]byte{[]byte("Genesis block")}
}

// GenNewReq: generate new request
// note: if the view leader is in new-view phase, send the request to the leader, or send it to next view leader
func GenNewReq(simulateNodes []*ofactory.Node, msg []string) {
	for i, v := range simulateNodes {
		if v.BasicHotstuff.IsLeader() {
			if v.BasicHotstuff.CurPhase == hstypes.NEW_VIEW || v.BasicHotstuff.CurPhase == hstypes.WAITING {
				simulateNodes[i].Requests = append(simulateNodes[i].Requests, common.String2ByteSlice(msg)...)
			} else {
				simulateNodes[(i+1)%len(simulateNodes)].Requests = append(simulateNodes[(i+1)%len(simulateNodes)].Requests, common.String2ByteSlice(msg)...)
			}
			break
		}
	}
}

// AutoGenNewReq: in basic-hotstuff, auto-generate requests
// params:
// - params: slice of parameters
// -- 1.count: Number of requests
// -- 2.reqNum: The number of transactions per request
// -- 3.length: Length of each request
func AutoGenNewReq(simulateNodes []*ofactory.Node, params []string) {

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
	for i := 0; i < paramInt[0]; i++ {
		GenNewReq(simulateNodes, common.GenerateSecureRandomStringSlice(paramInt[1], paramInt[2]))
		time.Sleep(time.Millisecond * 10)
	}
}

// CheckBlkInfo: check block info about hash
func CheckBlkInfo(simulateNodes []*ofactory.Node) {
	height, err := blockchain.GetBlockHeight("E:/MyOwnDoc/Project/GoProject/src/DeltaChain/BCData/" + simulateNodes[0].NodeID.ID.Name)
	if err != nil {
		fmt.Println(err)
	}
	for i := 0; i <= int(height); i++ {
		blk, _ := simulateNodes[0].BlkStore.ReadBlock("E:/MyOwnDoc/Project/GoProject/src/DeltaChain/BCData/"+simulateNodes[0].NodeID.ID.Name+"/", i)
		fmt.Println("Height " + strconv.Itoa(i))
		fmt.Println("PreBlock", blk.BlkHdr.PreBlkHash)
		fmt.Println("CurBlock", blk.Hash())
	}
}

// GenEmptyQC: generate a valid empyt QC by the qcType
func GenEmptyQC(simulateNodes []*ofactory.Node, qcType hstypes.StateType) *hstypes.QC {
	emptyQC := &hstypes.QC{
		QType:      qcType,
		ViewNumber: -1,
		HsNode: common.HsNode{
			CurHash:    merkle.EmptyHash(),
			ParentHash: merkle.EmptyHash(),
		},
	}
	partSign := make([][]byte, 0)
	for i := 0; i < len(simulateNodes); i++ {
		pS, err := simulateNodes[i].BasicHotstuff.ThresholdSigner.ThresholdSign(emptyQC.QC2SignMsgByte())
		if err == nil {
			partSign = append(partSign, pS)
		}
	}

	prepareQCSign, err := simulateNodes[0].BasicHotstuff.ThresholdSigner.CombineSig(emptyQC.QC2SignMsgByte(), partSign)

	if err == nil {
		emptyQC.Sign = prepareQCSign
		return emptyQC
	}
	return nil
}
