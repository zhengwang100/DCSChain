package orderer

import (
	"common"
	"encoding/json"
	hstypes "hotstuff/types"
	hs2types "hotstuff2/types"
	"message"
	ptypes "pbft/types"
)

// IsLeader: check whether self is leader
func (o *Orderer) IsLeader() bool {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		return o.BasicHotstuff.IsLeader()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		return o.ChainedHotstuff.IsLeader()
	case common.HOTSTUFF_2_PROTOCOL:
		return o.Hotstuff2.IsLeader()
	case common.PBFT:
		return o.PBFTConsensus.IsLeader()
	default:
		return false
	}
}

// IsWaitingReq: etects whether the orderer is in the state of waiting for a request
func (o *Orderer) IsWaitingReq() bool {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		return o.BasicHotstuff.CurPhase == hstypes.WAITING
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		return o.ChainedHotstuff.CurPhase == hstypes.WAITING
	case common.HOTSTUFF_2_PROTOCOL:
		return o.Hotstuff2.CurPhase == hs2types.NEW_PROPOSE
	case common.PBFT:
		return o.PBFTConsensus.CurPhase == ptypes.WAITING
	default:
		return false
	}
}

// GetLeaderName: get leader of current view name
func (o *Orderer) GetLeaderName() string {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		return o.BasicHotstuff.GetLeaderName()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		return o.ChainedHotstuff.GetLeaderName()
	case common.HOTSTUFF_2_PROTOCOL:
		return o.Hotstuff2.GetLeaderName()
	case common.PBFT:
		return o.PBFTConsensus.GetLeaderName()
	default:
		return ""
	}
}

// SendMsg: accepte the message returned in the consensus and send it
func (o *Orderer) SendMsg(payload interface{}) {
	serMsg := message.ServerMsg{
		SType: message.ORDER,
	}
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		// need to cast it with an assertion first in the case of basic hotstuff,
		payloadBMsg, ok := payload.(*hstypes.Msg)
		if ok {
			serMsg.SendServer = payloadBMsg.SendNode
			serMsg.ReciServer = payloadBMsg.ReciNode
			pBMsgJson, err := json.Marshal(payloadBMsg)
			if err != nil {
				o.BasicHotstuff.Logger.Println("[ERROR]", o.BasicHotstuff.GetNodeName(), err.Error())
			}
			serMsg.Payload = pBMsgJson

			o.SendChan <- serMsg
		} else {
			o.BasicHotstuff.Logger.Println("[ERROR]", o.BasicHotstuff.GetNodeName(), "assert basic-message error")
			return
		}
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		payloadCMsg, ok := payload.(*hstypes.CMsg)
		if ok {
			serMsg.SendServer = payloadCMsg.SendNode
			serMsg.ReciServer = payloadCMsg.ReciNode
			pBMsgJson, err := json.Marshal(payloadCMsg)
			if err != nil {
				o.ChainedHotstuff.Logger.Println("[ERROR]", o.ChainedHotstuff.GetNodeName(), err.Error())
			}
			serMsg.Payload = pBMsgJson
			o.SendChan <- serMsg
		} else {
			o.ChainedHotstuff.Logger.Println("[ERROR]", o.ChainedHotstuff.GetNodeName(), "assert chained-message error")
			return
		}
	case common.HOTSTUFF_2_PROTOCOL:
		payloadH2Msg, ok := payload.(*hs2types.H2Msg)
		if ok {
			serMsg.SendServer = payloadH2Msg.SendNode
			serMsg.ReciServer = payloadH2Msg.ReciNode
			pBMsgJson, err := json.Marshal(payloadH2Msg)
			if err != nil {
				o.Hotstuff2.Logger.Println("[ERROR]", o.Hotstuff2.GetNodeName(), err.Error())
			}
			serMsg.Payload = pBMsgJson
			o.SendChan <- serMsg
		} else {
			o.Hotstuff2.Logger.Println("[ERROR]", o.Hotstuff2.GetNodeName(), "assert chained-message error")
			return
		}
	case common.PBFT:
		payloadPMsg, ok := payload.(*ptypes.PMsg)
		if ok {
			serMsg.SendServer = payloadPMsg.SendNode
			serMsg.ReciServer = payloadPMsg.ReciNode
			pBMsgJson, err := json.Marshal(payloadPMsg)
			if err != nil {
				o.Hotstuff2.Logger.Println("[ERROR]", o.Hotstuff2.GetNodeName(), err.Error())
			}
			serMsg.Payload = pBMsgJson
			o.SendChan <- serMsg
		} else {
			o.Hotstuff2.Logger.Println("[ERROR]", o.Hotstuff2.GetNodeName(), "assert chained-message error")
			return
		}
	}
}
