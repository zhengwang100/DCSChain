package clientinfo

import "net"

// ClientInfo: client information stored on the server
type ClientInfo struct {
	Name string    // the name of client
	Addr string    // the ip address of client
	Pk   []byte    // the public key of client
	Conn *net.Conn // the tcp connect between the server and client
}
