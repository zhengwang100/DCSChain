package main

import (
	"bufio"
	common "common"
	"deltachain/core/client"
	"factory"
	"flag"
	"fmt"
	"log"
	"mgmt"
	"net"
	"os"
	"server"
	"strings"
	"test"
)

func main() {

	// debug.SetGCPercent(200)

	rolePtr := flag.String("r", "server", "server/client")
	portPtr := flag.String("po", "20000", "the port to listen")
	protocolPtr := flag.String("pr", "bh", "The protocol to use")
	pathPtr := flag.String("pa", "./BCData", "The protocol to use")
	nodePtr := flag.Int("n", 4, "The node number")

	// parse command line arguments
	flag.Parse()

	// gets the value of a command line argument
	role := *rolePtr
	port := *portPtr
	protocol := *protocolPtr
	path := *pathPtr
	node := *nodePtr

	// default protocol is basic-hotstuff
	pro := common.HOTSTUFF_PROTOCOL_BASIC

	if role == "server" {
		// generate a new logger for logging in main func
		mainLogger := *log.New(os.Stdout, "", 0)
		mainLogger.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

		// firstly generate new nodes and start the first chained round with command "Genesis block"
		switch protocol {
		case "bh":
			pro = common.HOTSTUFF_PROTOCOL_BASIC
		case "ch":
			pro = common.HOTSTUFF_PROTOCOL_CHAINED
		case "h2":
			pro = common.HOTSTUFF_2_PROTOCOL
		case "pbft":
			pro = common.PBFT
		default:
			fmt.Println("Input invalid")
		}
		simulateServers := factory.GenServers(node, path, pro, mgmt.BASIC)
		// mainLogger.Println(simulateServers)
		factory.ClearBlockInPath(simulateServers, path)
		mainLogger.Println("All nodes are started and ready", simulateServers[0].Orderer.ConsType)

		StartServerPort(port, simulateServers)

	} else if role == "client" {
		address := "127.0.0.1"

		client := client.NewClient(address, port)
		client.StartClient()
	} else {
		fmt.Println("Invalid mode. Please specify 'server' or 'client'.")
	}
}

// StartServerPort: the server starts port listening
// params:
// - port: the server start listening port
// - simulateServers: all nodes participating in the consensus
func StartServerPort(port string, simulateServers []*server.Server) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer ln.Close()

	fmt.Println("Server started on port", port)

	// whenever a node connects to the server, the server's request is processed
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}
		go handleClient(conn, simulateServers)
	}
}

// handleClient: the server handle the client connect
// params:
// - conn: the tcp connection between the server and the client
// - simulateServers: all nodes participating in the consensus
func handleClient(conn net.Conn, simulateServers []*server.Server) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

outerLoop:
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			break
		}
		fmt.Print("Received message:", msg)

		msg = strings.TrimRight(msg, "\r\n")
		input := strings.Split(msg, " ")

		// the input command calls the function to process
		switch input[0] {
		case "r":
			factory.GenNewReq(simulateServers, factory.SignCmd(common.StringSlice2TwoDimByteSlice(input[1:])))
		case "a":
			factory.AutoGenNewReq(simulateServers, input[1:])
		case "c":
			fmt.Println(len(simulateServers[1].NodeManager.NodesTable))
			// factory.CheckNodeInfo(simulateServers)
		case "b":
			count := factory.CheckBlkInfo(simulateServers)
			fmt.Println("all blocks contains commonds:", count)
			// p2p.Send(simulateServers[0].Clients["c_0"].Addr, append([]byte{byte(count)}, []byte("\n")...))
		case "j":
			test.NewServerJoin(&simulateServers)
		case "e":
			test.OriServerExit(&simulateServers, input[1:])
		case "q":
			factory.StopAll(simulateServers)
			fmt.Println("all server has stopped")
			break outerLoop
		default:
			// mainLogger.Println("Input invalid")
		}

		// Echo back to client
		// conn.Write([]byte("Server: " + msg))
	}
}
