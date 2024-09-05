package orderer

import (
	"bcrequest"
	"common"
)

// HandleReq: the orderer calls different HandleReq functions depending on its consensus type
func (o *Orderer) HandleReq(height int, preHash []byte, curHash []byte, req []bcrequest.BCRequest) {

	// when ReqStat is false, means the orderer stop the node
	if o.ReqState {

		switch o.ConsType {
		case common.HOTSTUFF_PROTOCOL_BASIC:
			// fmt.Println(height, preHash)
			o.BasicHotstuff.HandleReq(height, preHash, req)
		case common.HOTSTUFF_PROTOCOL_CHAINED:
			o.ChainedHotstuff.HandleReq(height, preHash, req)
		case common.HOTSTUFF_2_PROTOCOL:
			o.Hotstuff2.HandleReq(height, preHash, req)
		case common.PBFT:
			o.PBFTConsensus.HandleReq(height, preHash, curHash, req)
		default:
			return
		}
	}
}

// HandleMsg: the orderer calls different HandleMsg functions depending on its consensus type
func (o *Orderer) HandleMsg(msgJson []byte, pk []byte) {

	// when HandleState is false, means the orderer stop the node
	if o.HandleState {
		switch o.ConsType {
		case common.HOTSTUFF_PROTOCOL_BASIC:
			o.BasicHotstuff.HandleBMsg(msgJson, pk)
		case common.HOTSTUFF_PROTOCOL_CHAINED:
			o.ChainedHotstuff.HandleCMsg(msgJson)
		case common.HOTSTUFF_2_PROTOCOL:
			o.Hotstuff2.HandleH2Msg(msgJson)
		case common.PBFT:
			o.PBFTConsensus.HandlePMsg(msgJson)
		}
	}
}
