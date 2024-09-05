package factory

import (
	"blockchain"
	"common"
	"fmt"
	"server"
)

// CheckBlkInfo: counts the number of all commands in the block
func CheckBlkInfo(simulateNodes []*server.Server) int {
	height, err := blockchain.GetBlockHeight("./BCData/" + simulateNodes[0].ServerID.ID.Name + "/")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("block height is", height)
	count := 0
	for i := 0; i <= int(height); i++ {
		blk, _ := simulateNodes[0].BlkStore.ReadBlock("./BCData/"+simulateNodes[0].ServerID.ID.Name+"/", i)
		// fmt.Println("Height " + strconv.Itoa(i))
		// fmt.Println("PreBlock", blk.BlkHdr.PreBlkHash)
		// fmt.Println("CurBlock", blk.Hash())
		count += len(blk.BlkData.Trans)
		fmt.Println("block", i, len(blk.BlkData.Trans), common.StringSlice2TwoDimByteSlice(blk.BlkData.Trans)[0][:10])
	}
	return count
}
