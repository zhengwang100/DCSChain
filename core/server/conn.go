package server

import (
	"bufio"
	"fmt"
	"net"
)

// StartServer: start the server
func (s *Server) StartServer() {
	ln, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer ln.Close()

	fmt.Println("Server started on port", s.Port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}
		go handleClient(conn)
	}
}

// handleClient: handle the client and commands
func handleClient(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	reader := bufio.NewReader(conn)

	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			break
		}
		fmt.Print("Received message:", msg)

		// Echo back to client
		conn.Write([]byte("Server: " + msg))
	}
}
