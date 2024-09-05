package basictest

import (
	mysm4 "bccrypto/encrypt_sm4"
	"blockchain"
	"bufio"
	"common"
	"encoding/json"
	"fmt"
	hstypes "hotstuff/types"
	"log"
	"merkle"
	"mgmt"
	"ofactory"
	"os"
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
		case "q":
			break outerLoop
		case "c":
			fmt.Println(len(simulateNodes[1].NodeManager.NodesTable))
			// CheckNodeInfo(simulateNodes)
		case "b":
			CheckBlkInfo(simulateNodes)
		case "j":
			NewNodeJoin(&simulateNodes)
		case "e":
			NodeExit(&simulateNodes, input[1:])
		default:
			mainLogger.Println("Input invalid")
		}
	}
}

// GenNodes: generate the specified number of new nodes
// params:
// -nodeNum:	the number of node, node number
// -path:		the path fo block storage
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

// CheckNodeInfo: print the chained node information
// params:
// simulateNodes: the slice of nodes in system
func CheckNodeInfo(simulateNodes []*ofactory.Node) {
	for i := range simulateNodes {
		fmt.Println(simulateNodes[i].NodeID.ID.Name, ":", simulateNodes[i])
	}
}

// GenChainedFirstRound: generate the first request with command 'Genesis block' and start the first consensus round
// params:
// simulateNodes: the slice of nodes in system
func GenFirstRound(simulateNodes []*ofactory.Node) {
	dirPath := "./BCData/"
	if len(simulateNodes) == 0 {
		return
	}
	// update the first leader's request
	simulateNodes[0].Requests = [][]byte{[]byte("Genesis block")}

	emptyQC := GenEmptyQC(simulateNodes, hstypes.PREPARE)
	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].NodeID.ID.Name + "/")

		// generate the new message
		msg := hstypes.Msg{
			MType:      hstypes.NEW_VIEW,
			ViewNumber: 0,
			SendNode:   "r_" + strconv.Itoa(i),
			HsNode: common.HsNode{
				CurHash:    merkle.EmptyHash(),
				ParentHash: merkle.EmptyHash(),
			},
			Justify:    *emptyQC,
			PartialSig: nil,
		}

		// fmt.Println((simulateNodes[i]).NodeManager.NodesChannel["r_0"])
		// encode the message and send it
		msgJson, err := json.Marshal(msg)
		if err == nil {
			(simulateNodes[i]).NodeManager.NodesChannel["r_0"] <- msgJson
		}
	}
}

// GenNewReq: generate new request
// note: if the view leader is in new-view phase, send the request to the leader, or send it to next view leader
// params:
// -simulateNodes:	the slice of nodes in system
// -msg:			the requests, omit empty
func GenNewReq(simulateNodes []*ofactory.Node, msg []string) {
	leaderName := GetLeader(simulateNodes)
	if leaderName == "" {
		return
	}
	for i, v := range simulateNodes {
		if v.NodeID.ID.Name == leaderName {

			// fmt.Println(v.NodeID.ID.Name, v.BasicHotstuff.View.ViewNumber, v.BasicHotstuff.CurPhase)

			if v.BasicHotstuff.CurPhase == hstypes.NEW_VIEW || v.BasicHotstuff.CurPhase == hstypes.WAITING {
				simulateNodes[i].Requests = append(simulateNodes[i].Requests, common.String2ByteSlice(msg)...)
			} else {
				simulateNodes[(i+1)%len(simulateNodes)].Requests = append(simulateNodes[(i+1)%len(simulateNodes)].Requests, common.String2ByteSlice(msg)...)
			}
			break
		}
	}
}

// AutoGenChainedNewReq: in basic-hotstuff, auto-generate requests
// params:
// -simulateNodes: the slice of nodes in system
// -params[]:
// --1.count: Number of requests
// --2.reqNum: The number of transactions per request
// --3.length: Length of each request
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
// params:
// -simulateNodes: the slice of nodes in system
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
// params:
// -simulateNodes: 	the slice of nodes in system
// -qcType:			what qc needs to be generated in what state
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

// NewNodeJoin: create a new node and have it initiate a join request to the system
// params:
// -simulateNodes: the slice of nodes in system
func NewNodeJoin(simulateNodes *[]*ofactory.Node) {

	// stop all nodes
	// StopAllNodes(simulateNodes)

	nodeName := "r_" + strconv.Itoa(len(*simulateNodes))
	newNode, err := ofactory.NewNode(len(*simulateNodes), common.HOTSTUFF_PROTOCOL_BASIC, "./BCData")
	if err != nil {
		return
	}

	// update new node information
	newNode.NodeManager.NodesChannel[nodeName] = newNode.NodeID.Address
	newNode.BasicHotstuff.ForwardChan = newNode.NodeManager.NodesChannel[nodeName]

	newNode.NodeManager.NewNode.Name = newNode.NodeID.ID.Name
	newNode.NodeManager.NewNode.NodeKey.Sm2PubKey = newNode.NodeID.ID.PubKey
	newNode.NodeManager.NewNode.Chan = newNode.NodeID.Address

	// add PubKey of nodes in system
	for _, node := range *simulateNodes {
		sm4PK := mysm4.GenerateKey()
		newNode.NodeManager.NodesChannel[node.NodeID.ID.Name] = node.NodeID.Address
		newNode.NodeManager.NodesTable[node.NodeID.ID.Name] = mgmt.NodeKey{
			Name:      node.NodeID.ID.Name,
			Sm2PubKey: node.NodeID.ID.PubKey,
			Sm4Key:    sm4PK,
		}
	}

	// start two process to handle the message and handle requests
	go newNode.HandleMsg(newNode.NodeManager.NodesChannel[nodeName])
	go newNode.HandleReq()

	go newNode.StartBCNodeJoin(*simulateNodes)
	// joinMsg := hstypes.Msg{}

	// add new node to the slice of simulated nodes
	*simulateNodes = append(*simulateNodes, newNode)
	// fmt.Println("In", len(newNode.NodeManager.NodesTable))

	// UpdateSigners: update simulateServers' orderer signer
	go func(t int) {
		time.Sleep(time.Duration(t) * time.Millisecond)
		UpdateSigners(*simulateNodes)
	}(100)
}

// UpdateSigners: update simulateServers' orderer signer
// params:
// -simulateNodes: the slice of nodes in systems
func UpdateSigners(simulateNodes []*ofactory.Node) {
	nodeNum := len(simulateNodes)
	if simulateNodes[0].ConsType == common.PBFT {
		for i := 0; i < nodeNum-1; i++ {
			simulateNodes[nodeNum].PBFTConsensus.Signer.Pks[simulateNodes[i].NodeID.ID.Name] = simulateNodes[i].PBFTConsensus.Signer.Pk
		}
	} else {
		newsigners := tss.NewSigners(nodeNum, (nodeNum-1)/3*2+1)
		for i := 0; i < nodeNum; i++ {
			simulateNodes[i].BasicHotstuff.ThresholdSigner = newsigners[i]
			simulateNodes[i].BasicHotstuff.Logger.Println("[SIGNER_UPDATE]:", (simulateNodes)[i].NodeID.ID.Name, "succeed")
		}
	}
}

// StopAllNodes: stop all nodes in system
// params:
// -simulateNodes: the slice of nodes in system
func StopAllNodes(simulateNodes []*ofactory.Node) {
	for _, node := range simulateNodes {
		node.HandleState = false
		node.ReqState = false
		node.BasicHotstuff.ViewTimer.Stop()
	}
}

// params:
// -simulateNodes: the slice of nodes in system
func RestartAllNodes(simulateNodes []*ofactory.Node) {
	for _, node := range simulateNodes {
		node.HandleState = true
		node.ReqState = true

		go node.HandleMsg(node.NodeManager.NodesChannel[node.NodeID.ID.Name])
		go node.HandleReq()
	}
}

// params:
// -simulateNodes: the slice of nodes in system
func GetLeader(simulateNodes []*ofactory.Node) string {
	leaders := make(map[string]int, 2)
	leaders["r_"+strconv.Itoa(len(simulateNodes))] = 0
	leaders["r_"+strconv.Itoa(len(simulateNodes)-1)] = 0
	for _, n := range simulateNodes {
		leaders[n.BasicHotstuff.GetLeaderName()] += 1
	}
	for name, count := range leaders {
		if count > (len(simulateNodes)-1)/3*1 {
			return name
		}
	}
	return ""
}

// params:
// -simulateNodes: 	the slice of nodes in system
// -id:				the id of exit node
func NodeExit(simulateNodes *[]*ofactory.Node, id []string) {
	if len(id) == 0 {
		fmt.Println("None param")
		return
	}
	name := "r_" + id[0]
	for index, n := range *simulateNodes {
		if n.NodeID.ID.Name == name {
			n.StartBCNodeExit()

			*simulateNodes = append((*simulateNodes)[:index], (*simulateNodes)[index+1:]...)

			break
		}
	}
	// fmt.Println("In", len(newNode.NodeManager.NodesTable))
	go func(t int) {
		time.Sleep(time.Duration(t) * time.Millisecond)
		UpdateSigners(*simulateNodes)
	}(100)
}
