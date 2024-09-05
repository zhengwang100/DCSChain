package bcmanager_test

import (
	"bcmanager"
	"crypto/rand"
	"fmt"
	"mgmt"
	"testing"

	"github.com/xlcetc/cryptogm/sm/sm2"
)

// TestEmbedding: test whether a nodetable is properly bound to the node manager
func TestEmbedding(t *testing.T) {
	name := "r_1"
	_, pk, _ := sm2.Sm2KeyGen(rand.Reader)
	nodesTable := map[string]mgmt.NodeKey{
		name: {
			Name:      name,
			Sm2PubKey: pk,
		},
	}
	ttt := bcmanager.NewNodeManager(1, nodesTable, nil)

	fmt.Println(ttt)

	tt := []string{"123", "213"}
	fmt.Println(tt, tt[:1], tt[2:])
}
