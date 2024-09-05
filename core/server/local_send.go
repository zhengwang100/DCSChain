package server

import (
	"fmt"
	"local"
	"message"
	"p2p"

	"github.com/xlcetc/cryptogm/sm/sm2"
	"github.com/xlcetc/cryptogm/sm/sm3"
)

// SendMsg: convert message to json and send it
func (s *Server) SendMsg(msg message.ServerMsg) {

	// sign the message
	h := sm3.SumSM3(msg.Payload)
	sign, err := sm2.Sm2Sign(s.ServerID.PrivateKey, s.ServerID.ID.PubKey, h[:])
	if err != nil {
		s.Logger.Println("Server sign message error", err)
		return
	}

	msg.Sign = sign
	msgJson, err := message.EncodeMsg(msg)
	if err == nil {

		// simulate network delay
		// time.Sleep(10 * time.Millisecond)

		switch msg.ReciServer {
		case "Broadcast":
			local.Broadcast(s.NodeManager.NodesChannel, msgJson, s.ServerID.ID.Name)
			// s.Logger.Println("[Broadcast]", s.ServerID.ID.Name+" ->", s.NodeManager.GetNodeNames())

		case "Gossip":
			local.Gossip(s.NodeManager.NodesChannel, msgJson, msg.SendServer)
			// s.Logger.Println("[Gossip]", s.ServerID.ID.Name+" ->", s.NodeManager.GetOtherNodeNames())

		case "Client":
			// p2p.SendUdp(s.ClientAddr, append(msgJson, []byte("\n")...))
			conn := p2p.Send(s.Clients["c_0"].Conn, s.Clients["c_0"].Addr, append(msgJson, []byte("\n")...))
			if conn != nil {
				s.Clients["c_0"].Conn = conn
			}
			// s.Logger.Println("[SendClient]", s.ServerID.ID.Name+" ->", "Client", len(msgJson))

		case s.NodeManager.NewNode.Name:

			local.Fixedcast(s.NodeManager.NewNode.Chan, msgJson)
			// s.Logger.Println("[Fixedcast]", s.ServerID.ID.Name+" ->", msg.ReciServer)

		default:
			local.Unicast(s.NodeManager.NodesChannel, msgJson, msg.ReciServer, s.ServerID.ID.Name)
			// s.Logger.Println("[Unicast]", s.ServerID.ID.Name+" ->", msg.ReciServer)
		}
	} else {
		fmt.Println(err)
	}
}
