package factory

import (
	"bcrequest"
	common "common"
	"server"
	"ssm2"
)

// GenChainedFirstRound: generate the first request with command 'Genesis block'
// and start the first consensus round
// params:
// - simulateServers: the slice of nodes in system
// - path:			  the path of block storage
func GenFirstRound(simulateNodes []*server.Server, path string) {
	dirPath := path + "/"
	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].ServerID.ID.Name + "/")
	}

	signer := ssm2.Signer{}
	signer.Sk = signer.GetSKFromFile()
	signer.Pk = signer.GetPKFromFile()
	cmd := []byte("Genesis block")
	sign := signer.Sign(cmd)
	// update the first leader's request
	simulateNodes[0].Requests = []bcrequest.BCRequest{{
		Id:   "Client",
		Cmd:  cmd,
		Sign: sign,
	},
	}
	simulateNodes[0].Orderer.ReqFlagChan <- true
}

// ClearBlockInPath: clear block in path
// params:
// - simulateServers: the slice of nodes in system
// - path:			  the path of block storage
func ClearBlockInPath(simulateNodes []*server.Server, path string) {
	dirPath := path + "/"
	for i := 0; i < len(simulateNodes); i++ {

		// remove self past files, and choose not to delete it as required
		common.RemoveAllFilesAndDirs(dirPath + simulateNodes[i].ServerID.ID.Name + "/")
	}
}
