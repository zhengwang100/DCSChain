package factory

import (
	common "common"
	"ssm2"
	"tss"
)

// GenSigners: generate n signers that satisfy the condition of threshold 2f+1
// or the sm2 signer for pbft
// params:
// - consType:the consensus protocol type
// - nodeNum: the node number
func GenSigners(consType common.ConsensusType, nodeNum int) []interface{} {
	newSignes := make([]interface{}, 0)
	if consType == common.PBFT {
		// pbft protocol use ssm2
		ssm2Signers := ssm2.NewSigners(nodeNum)
		for _, v := range ssm2Signers {
			newSignes = append(newSignes, v)
		}
	} else {
		// other protocol use BLS Threshold signature
		tssSigners := tss.NewSigners(nodeNum, (nodeNum-1)/3*2+1)
		for _, v := range tssSigners {
			newSignes = append(newSignes, v)
		}
	}
	return newSignes
}
