package server

import (
	"bcmanager"
	"bcrequest"
	"blockchain"
	ci "clientinfo"
	common "common"
	"config"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"identity"
	"log"
	"message"
	"mgmt"
	"orderer"
	"os"
	"ssm2"
	"strconv"
	"sync"

	"github.com/xlcetc/cryptogm/sm/sm2"
	"github.com/xlcetc/cryptogm/sm/sm3"
)

// Server is the system node which is the main unit
type Server struct {
	ServerID identity.PrivID           // the only identity the node in system
	Port     string                    // open port monitored by the server
	Clients  map[string]*ci.ClientInfo // the client info

	Orderer orderer.Orderer // the orderer unit for consistence by consensus

	NMType      mgmt.NodeManagerType  // the node manager type, now is basic
	NodeManager bcmanager.NodeManager // the node manager

	SendChan     chan message.ServerMsg // the channel within the server that receives all messages that need to be sent
	Requests     []bcrequest.BCRequest  // the server recieved requests with signatures
	RequestsLock sync.Mutex
	BatchSize    int                   // the number of request within a block
	BlkStore     blockchain.BlockStore // the blockchain storage, which is responsible for blockchain-related storage queries, etc
	Logger       log.Logger            `json:"logger"` // logger responsible for logging
}

// NewServer: create a new server according to different parameters
// params:
// id: 			the unique identification of the server
// nodeNum: 	the number of nodes in the system
// pathe: 		the path of block storage
// consType: 	the consensus protocol type selected by the server
// nmType: 		the node manager type selected by the server
// signer:
// nodesChannel:
// return a server instance and error
func NewServer(id int, nodeNum int, path string, consType common.ConsensusType, nmType mgmt.NodeManagerType, signer interface{}, nodesChannel map[string]chan []byte) (*Server, error) {

	// get the server name
	name := "r_" + strconv.Itoa(id)

	// generate node private key and public key
	sk, pk, err := sm2.Sm2KeyGen(rand.Reader)
	if err != nil {
		return nil, err
	}

	// init nodesTable and add self
	nodesTable := map[string]mgmt.NodeKey{
		name: {
			Name:      name,
			Sm2PubKey: pk,
		},
	}

	// read client key from file only for test
	cpk := ssm2.ReadKey("./config/client/public.pem")

	if len(cpk) == 0 {
		cpk = ssm2.ReadKey("../../config/client/public.pem")
	}

	clientInfo := map[string]*ci.ClientInfo{
		"c_0": {
			Name: "c_0",
			Addr: "127.0.0.1:30000",
			Pk:   cpk,
			Conn: nil,
		},
	}

	// create the new server instance
	newServer := &Server{
		ServerID: identity.PrivID{
			ID: identity.PubID{
				Name:   name,
				PubKey: pk,
			},
			Address:    make(chan []byte, 1024),
			PrivateKey: sk,
		},
		Port:      strconv.Itoa(id + 20001),
		Clients:   clientInfo,
		NMType:    nmType,
		SendChan:  make(chan message.ServerMsg, 128),
		Logger:    *log.New(os.Stdout, "", 0),
		Requests:  make([]bcrequest.BCRequest, 0),
		BatchSize: config.BatchSize,
	}

	// init node manager
	newServer.InitNodeManager(nmType, id, nodesTable, nodesChannel)

	// init consensus
	newServer.InitConsensus(consType, id, nodeNum, path, newServer.SendChan, signer)
	// newServer.BlkStore = newServer.Orderer.BasicHotstuff.BlkStore

	// set the log format
	newServer.Logger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	return newServer, nil
}

// RouteServerMsg: route recieved server message from different channel
// params:
// ch: channel for recieving server message
func (s *Server) RouteServerMsg(ch chan []byte) {
	for {
		select {
		case msgJson := <-ch:

			msg := message.DecodeMsg(msgJson)
			if msg == nil {
				continue
			}

			// verify the message signature
			h := sm3.SumSM3(msg.Payload)
			if !sm2.Sm2Verify(msg.Sign, s.NodeManager.NodesTable[msg.SendServer].Sm2PubKey, h[:]) {
				s.Logger.Println("Message verify sign error", msg.SendServer)
			}

			switch msg.SType {
			case message.REQUEST:
				s.ValidateAndHandleReq(msg.Payload)
			case message.NODEMGMT:
				s.SubmitMsg2NodeManager(msg.Payload)
			case message.ORDER:
				s.SubmitMsg2Consensus(msg.Payload)
			default:
				fmt.Println("Server message type is unknown type!")
			}
		case serMsg := <-s.SendChan:

			// sendChan is internal channel, which is messages submitted by other components to the server to be sent
			s.SendMsg(serMsg)
		}
	}
}

// ValidateAndHandleReq: the server handle requests to order if the order is leader and waiting for new request
// the func maybe is to validate the validation of requests, but now is directly submit the request to orderer for handle it
// params:
// req: requests recieved
func (s *Server) ValidateAndHandleReq(payload []byte) {
	req := bcrequest.BCRequest{}
	json.Unmarshal(payload, &req)

	// validate the validation of requests here, but now is none and directly append
	s.Requests = append(s.Requests, req)

	// if the server is waiting requests and submit
	if s.Orderer.IsWaitingReq() {
		s.Orderer.HandleReq(s.BlkStore.Height, s.BlkStore.PreBlkHash, s.BlkStore.CurBlkHash, s.Requests)
	}
}

// InitNodeManager: init consensus, until now only basic node-manager
// params:
// nmType:			the node manager type selected by the server
// id: 				the unique identification of the server
// nodesTable: 		the all known node information table in system
// nodesChannel: 	the channels table of all nodes
func (s *Server) InitNodeManager(nmType mgmt.NodeManagerType, id int, nodesTable map[string]mgmt.NodeKey, nodesChannel map[string]chan []byte) {
	switch nmType {
	case mgmt.BASIC:
		s.NodeManager = *bcmanager.NewNodeManager(id, nodesTable, nodesChannel)
	default:
		fmt.Println("NodeManager type is unknown type!")
	}
}

// InitConsensus: init consensus, include basic-hotstuff, chained-hotstuff, hotstuff-2 and PBFT
// params:
// - consType:		the consensus protocol type selected by the server
// - id: 			the unique identification of the server
// - nodeNum:		the number of nodes in system
// - path:			the path of block storage
// - sendChan: 	the channel within the server that receives all messages that need to be sent
// - signer: 	the channels table of all nodes
func (s *Server) InitConsensus(consType common.ConsensusType, id int, nodeNum int, path string, sendChan chan message.ServerMsg, signer interface{}) {
	s.Orderer.InitConsensus(consType, id, nodeNum, path, sendChan, signer)
}

// SubmitMsg2NodeManager: submit message to node manager
func (s *Server) SubmitMsg2NodeManager(msg []byte) {
	switch s.NMType {
	case mgmt.BASIC:
		s.HandleNodeManagerMsg(msg)
	}
}

// SubmitMsg2Consensus: submit message to consensus
func (s *Server) SubmitMsg2Consensus(msg []byte) {
	s.Orderer.HandleMsg(msg, s.Clients["c_0"].Pk)
}

// GetNodeNames: get node names from NodesChannel
// return all nodes name
func (s *Server) GetNodeNames() []string {
	return s.NodeManager.GetNodeNames()
}

// GetNodeNames: get node except itself names from NodesChannel
// return all nodes name except self
func (s *Server) GetOtherNodeNames() []string {
	return s.NodeManager.GetOtherNodeNames()
}

// Server verify reqests form client
func (s *Server) VerifyReqs() bool {
	length := len(s.Requests)
	if length == 0 {
		s.Logger.Println("[Error]: requests length is zero")
		return false
	}

	// for i := 0; i < length; i++ {
	// 	if !sm2.Sm2Verify(s.Requests[i].Sign, s.Clients["c_0"].Pk, s.Requests[i].Cmd) {
	// 		s.Logger.Println("[Error]: requests sign verify error")
	// 		return false
	// 	}
	// }

	return true
}
