package local

import (
	"time"
)

// Broadcast: send message to all nodes include itself
func Broadcast(nodeschan map[string]chan []byte, msg []byte, sendName string) {
	for _, nodeChan := range nodeschan {
		msgCopied := make([]byte, len(msg))
		copy(msgCopied, msg)
		nodeChan <- msgCopied
	}
}

func Unicast(nodeschan map[string]chan []byte, msg []byte, reciName string, sendName string) {
	nodeschan[reciName] <- msg
}

// Fixdcast: send to a fixed channel
func Fixedcast(ch chan []byte, msg []byte) {
	ch <- msg
}

// Gossip: send message to all nodes except itself
func Gossip(nodeschan map[string]chan []byte, msg []byte, sendName string) {
	for name, nodeChan := range nodeschan {
		if name == sendName {
			continue
		}
		msgCopied := make([]byte, len(msg))
		copy(msgCopied, msg)
		nodeChan <- msgCopied
	}
}

// GetTimestamp: get current timestamp
func GetTimestamp() int64 {
	now := time.Now()
	return now.Unix()
}
