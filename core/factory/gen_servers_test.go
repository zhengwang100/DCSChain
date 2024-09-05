package factory_test

import (
	common "common"
	"encoding/json"
	"factory"
	hstypes "hotstuff/types"
	"message"
	"mgmt"
	"testing"
	"time"
)

// TestGenServer: test func GenServer
func TestGenServer(t *testing.T) {
	testServers := factory.GenServers(4, "./BCData", common.HOTSTUFF_PROTOCOL_BASIC, mgmt.BASIC)

	testBMsg := hstypes.Msg{
		MType:  hstypes.NEW_VIEW,
		NMType: hstypes.SYNC,
	}
	testPayload, _ := json.Marshal(testBMsg)
	testMsg := message.ServerMsg{
		SType:      message.ORDER,
		SendServer: "test",
		ReciServer: "test",
		Payload:    testPayload,
	}

	testMJson, _ := message.EncodeMsg(testMsg)

	for _, s := range testServers {
		s.NodeManager.NodesChannel[s.ServerID.ID.Name] <- testMJson
	}

	time.Sleep(5 * time.Second)
}
