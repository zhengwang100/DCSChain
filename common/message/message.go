package message

import (
	"encoding/json"
)

// ServerMsg: the message between two servers
type ServerMsg struct {
	SType      MsgType // the type of message, REQUEST/NODEMGMT/ORDER
	SendServer string  // the server that sends the message
	ReciServer string  // the server that recieves the message
	Sign       []byte
	Payload    []byte // the payload that this message carries, such as a consensus message
}

// MsgType: the type of  ServerMsg
type MsgType uint8

const (
	REQUEST  MsgType = iota // client request message
	NODEMGMT                // message indicating that a node applies for joining or exiting
	ORDER                   // consensus message for orderer
)

// EncodeMsg: encode the serverMsg
func EncodeMsg(sMsg ServerMsg) ([]byte, error) {
	return json.Marshal(sMsg)
}

// DecodeMsg: decode the serverMsg
func DecodeMsg(msgJson []byte) *ServerMsg {
	res := &ServerMsg{}
	err := json.Unmarshal(msgJson, res)
	if err == nil {
		return res
	}
	return nil
}
