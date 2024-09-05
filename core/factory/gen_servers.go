package factory

import (
	mysm4 "bccrypto/encrypt_sm4"
	common "common"
	"fmt"
	"mgmt"
	"server"
	"strconv"
)

// GenServers: generate servers
// params:
// nodeNum: 	the number of nodes in the system
// pathe: 		the path of block storage
// consType: 	the consensus protocol type selected by the server
// nmType: 	the node manager type selected by the server
// return a slice of nodeNum server instances
func GenServers(nodeNum int, path string, consType common.ConsensusType, nmType mgmt.NodeManagerType) []*server.Server {

	// declare simulate nodes and channel belong to node
	// the initial generated nodes all use the same nodesChannel
	var simulateNodes []*server.Server
	nodesChannel := make(map[string]chan []byte)

	// generate n signers that satisfy the condition of threshold 2f+1 or the sm2 signer for pbft
	newSigners := GenSigners(consType, nodeNum)
	for i := 0; i < nodeNum; i++ {
		nodeName := "r_" + strconv.Itoa(i)
		newNode, err := server.NewServer(i, nodeNum, path, consType, mgmt.BASIC, newSigners[i], nodesChannel)
		if err != nil {
			fmt.Println("error:", err)
		} else {
			simulateNodes = append(simulateNodes, newNode)
			nodesChannel[nodeName] = newNode.ServerID.Address
		}

		// start two process to handle the message and handle requests
		go simulateNodes[i].RouteServerMsg(simulateNodes[i].NodeManager.NodesChannel[nodeName])
		go simulateNodes[i].HandleReq()
		// go simulateNodes[i].StartServer()
	}

	// update nodes' route table that record other node information
	for i := 0; i < nodeNum; i++ {
		for j := 0; j < nodeNum; j++ {
			if j != i {
				simulateNodes[i].NodeManager.NodesTable[simulateNodes[j].ServerID.ID.Name] = mgmt.NodeKey{
					Name:      simulateNodes[j].ServerID.ID.Name,
					Sm2PubKey: simulateNodes[j].ServerID.ID.PubKey,
				}
				if j > i {
					sm4PK := mysm4.GenerateKey()
					simulateNodes[i].NodeManager.UpdateSm4Key(simulateNodes[j].ServerID.ID.Name, sm4PK)
					simulateNodes[j].NodeManager.UpdateSm4Key(simulateNodes[i].ServerID.ID.Name, sm4PK)
				}
			}
		}
	}

	return simulateNodes
}
