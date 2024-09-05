package p2p

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// Send: send the message by TCP
// params:
// - conn: the TCP connection for which the message was sent
// - addr: if a connection has not been established, you need to create a link and send a message based on that address
func Send(conn *net.Conn, addr string, msg []byte) *net.Conn {
	if conn != nil {
		// if the link is not empty, it is sent directly
		(*conn).Write(msg)
		return nil
	} else {
		// if the link is empty, create a connection and send it
		newConn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Println("Error connecting to server:", err.Error())
			return nil
		}
		newConn.Write(msg)

		// return link to the server
		return &newConn
	}
}

// SendUdp: send the message by UDP
// - addr: the destination address of the message
// - msg: messages that need to be sent
func SendUdp(addr string, msg []byte) {
	// define destination address
	targetAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	// creating a UDP connection
	conn, err := net.DialUDP("udp", nil, targetAddr)
	if err != nil {
		fmt.Println("Error connecting to UDP server:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write(msg)
	if err != nil {
		fmt.Println("Error sending UDP message:", err)
		return
	}
}

// CheckTCP: check whether the tcp connection is possible
func CheckTCP(port int) bool {
	address := "localhost:" + strconv.Itoa(port)

	// attempting to connect to a TCP port
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		fmt.Printf("TCP port %d is closed\n", port)
		return false
	}
	defer conn.Close()

	return true
}

// CheckUDP: check whether the udp connection is possible
func CheckUDP(port int) bool {
	address := "localhost:" + strconv.Itoa(port)

	// attempt to connect to a UDP port
	conn, err := net.DialTimeout("udp", address, time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}
