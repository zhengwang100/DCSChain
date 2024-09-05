package factory

import (
	mysm4 "bccrypto/encrypt_sm4"
	common "common"
	"mgmt"
	"server"
	"strconv"
	"time"
	"tss"
)

// NewServerJoin: create a new node and have it initiate a join request to the system
// params:
// simulateServers: the slice of nodes in system
func NewBHServerJoin(simulateServers *[]*server.Server) {

	// create new node
	nodeName := "r_" + strconv.Itoa(len(*simulateServers))
	tSigner := tss.NewSigners(1, 1)
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

	// update new node information
	newServer.NodeManager.NodesChannel[nodeName] = newServer.ServerID.Address

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
	// joinMsg := hstypes.Msg{}

	// add new node to the slice of simulated nodes
	*simulateServers = append(*simulateServers, newServer)
	// fmt.Println("In", len(newServer.NodeManager.NodesTable))
	go func(t int) {
		time.Sleep(time.Duration(t) * time.Millisecond)
		UpdateSigners(*simulateServers)
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
	// 	simulateServers[i].Orderer.PBFTConsensus.ThresholdSigner = newsigners[i]
	// 	simulateServers[i].Orderer.PBFTConsensus.Logger.Println("[SIGNER_UPDATE]:", (simulateServers)[i].ServerID.ID.Name, "succeed")
	// }
	default:

	}
}
