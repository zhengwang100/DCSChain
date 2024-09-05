package ofactory_test

import (
	"bufio"
	"common"
	"encoding/json"
	"fmt"
	bhstypes "hotstuff/types"
	"ofactory"
	"os"
	"reflect"
	"testing"
)

// TestNewNode: test create a new node
func TestNewNode(t *testing.T) {
	testNode, err := ofactory.NewNode(-3, common.HOTSTUFF_PROTOCOL_BASIC, "./BCData")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(testNode.NodeID.ID.Name)
		fmt.Println(testNode.NodeID.PrivateKey)
		fmt.Println(testNode.NodeID.ID.PubKey)
		fmt.Println(reflect.TypeOf(testNode.Cons))

		// if bhs, ok := testNode.Cons.(core.BCHotstuff); ok {
		// fmt.Println( bhs.GetNodeName() == bhs.Leader)
		// }
	}

}

// TestSign: test the function of sgin of node
func TestSign(t *testing.T) {
	testNode, err := ofactory.NewNode(-2, common.HOTSTUFF_PROTOCOL_BASIC, "./BCData")
	if err != nil {
		fmt.Println(err)
	} else {
		msg := []byte{1, 2, 3}
		sign, err := testNode.Sign(msg)
		fmt.Println(sign)
		fmt.Print(err)
		res_ver := testNode.VerifySign(msg, testNode.NodeID.ID.PubKey, sign)
		fmt.Println(res_ver)
		msg2 := []byte{2, 3, 4}
		res_ver2 := testNode.VerifySign(msg2, testNode.NodeID.ID.PubKey, sign)
		fmt.Println(res_ver2)

	}
}

// TestChan: test node transfer messages
func TestChan(t *testing.T) {
	testNode, _ := ofactory.NewNode(-1, common.HOTSTUFF_PROTOCOL_BASIC, "./BCData")
	ch := make(chan []byte, 64)

	go testNode.HandleMsg(ch)

	for i := 0; i < 10; i++ {
		msgJson, _ := json.Marshal(bhstypes.Msg{
			MType:      bhstypes.NEW_VIEW,
			ViewNumber: 12,
			SendNode:   testNode.NodeID.ID.Name,
			Justify: bhstypes.QC{
				QType:      bhstypes.NEW_VIEW,
				ViewNumber: 12,
				HsNode:     common.HsNode{},
				Sign:       nil,
			},
			PartialSig: nil,
		})
		ch <- msgJson
	}
	// close(ch)
}

// TestQurom: test the correctness of qurom calculation
func TestQurom(t *testing.T) {
	for i := 3; i < 20; i++ {
		fmt.Println("n:", i, "f=", (i-1)/3, "2f:", (i-1)/3*2)
	}
}

// TestRead: test read commands from the command line
func TestRead(t *testing.T) {
	for {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("请输入内容:")

		// read line input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("读取输入时发生错误:", err)
			return
		}

		fmt.Println("你输入的内容是:", input)
	}
}
