package orderer

import (
	"common"
	"hotstuff/core"
	h2core "hotstuff2/core"
	"message"
	"mgmt"
	pcore "pbft/core"
	"ssm2"
	"tss"
)

// Orderer: the role responsible for consensus ordering in the system
type Orderer struct {
	ConsType        common.ConsensusType // the consensus protocol type selected by the server
	HandleState     bool                 // the flag of whether the consensus message can be accepted
	ReqState        bool                 // the flag of whether the request can be accepted
	ReqFlagChan     chan bool
	SendChan        chan message.ServerMsg // the channel that submits the message to the server that needs to be sent
	BasicHotstuff   *core.BCHotstuff       // the core of basic hotstuff consensus
	ChainedHotstuff *core.CHotstuff        // the core of chained hotstuff consensus
	Hotstuff2       *h2core.Hotstuff2      // the core of hotstuff-2 consensus
	PBFTConsensus   *pcore.PBFT            // the core of PBFT consensus
}

// InitConsensus: init consensus
// params:
// - consType:	the consensus protocol type
// - id:		the unique identification of the server
// - nodeNum:	the number of nodes in the system
// - path:the 	path of block storage
// - sendChan:	the channel within the server that receives all messages that need to be sent
// - signer:	the signer for signature
func (o *Orderer) InitConsensus(consType common.ConsensusType, id int, nodeNum int,
	path string, sendChan chan message.ServerMsg, signer interface{}) {

	// update order
	o.ConsType = consType
	o.SendChan = sendChan
	o.ReqFlagChan = make(chan bool, 1)
	o.HandleState = true
	o.ReqState = true
	switch consType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		tssSigner, ok := signer.(*tss.Signer)
		if !ok {
			panic("Signer type does not match!")
		}
		o.BasicHotstuff = core.NewBCHotstuff(5000, id, nodeNum, path, sendChan, tssSigner)

	case common.HOTSTUFF_PROTOCOL_CHAINED:
		tssSigner, ok := signer.(*tss.Signer)
		if !ok {
			panic("Signer type does not match!")
		}
		o.ChainedHotstuff = core.NewChainedHotstuff(2000, id, nodeNum, path, sendChan, tssSigner)

	case common.HOTSTUFF_2_PROTOCOL:
		tssSigner, ok := signer.(*tss.Signer)
		if !ok {
			panic("Signer type does not match!")
		}
		o.Hotstuff2 = h2core.NewHotstuff2(500, 2000, id, nodeNum, path, sendChan, tssSigner)

	case common.PBFT:
		sm2Signer, ok := signer.(*ssm2.Signer)
		if !ok {
			panic("Signer type does not match!")
		}
		o.PBFTConsensus = pcore.NewPBFT(10000, id, nodeNum, path, sendChan, sm2Signer)
	default:
		panic("Consensus type is unknown type!")
	}

	// if the orderer is leader, update its state to handle req
	if o.IsLeader() {
		o.InitLeader()
	}
}

// InitLeader: protocols need to initialize the leader
func (o *Orderer) InitLeader() {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.InitLeader()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.InitLeader()
	case common.HOTSTUFF_2_PROTOCOL:
		o.Hotstuff2.InitLeader()
	case common.PBFT:
		o.PBFTConsensus.InitLeader()
	default:
		return
	}
}

// FixLeader: when the threshold f of a newly added node needs to be updated,
// additional patching of the leader state is required
func (o *Orderer) FixLeader() {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.FixLeader()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.FixLeader()
	case common.HOTSTUFF_2_PROTOCOL:
	case common.PBFT:
	default:
		return
	}
}

// Stop: stop the leader, update the orderer HandleState and ReqState to false, stop timer
func (o *Orderer) Stop() {
	o.HandleState = false
	o.ReqState = false
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.ViewTimer.Stop()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.ViewTimer.Stop()
	case common.HOTSTUFF_2_PROTOCOL:
		o.Hotstuff2.PM.EnterTimer.Stop()
		o.Hotstuff2.PM.ViewTimer.Stop()
	case common.PBFT:
		o.PBFTConsensus.PTimer.Timer.Stop()
	}
}

// ResetState: clear the current messages, update the orderer HandleState and ReqState to true
func (o *Orderer) ResetState() {
	o.ClearCurrentRound()
	o.HandleState = true
	o.ReqState = true
}

// RestartCons: restart the consensus
func (o *Orderer) RestartCons() {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		msgReturn := o.BasicHotstuff.RestartBasicHotstuff()
		if msgReturn != nil {
			o.SendMsg(msgReturn)
		}
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		msgReturn := o.ChainedHotstuff.RestartChainedHotstuff()
		if msgReturn != nil {
			o.SendMsg(msgReturn)
		}
	case common.HOTSTUFF_2_PROTOCOL:
		// ignore to check QC in the first round after join or exit
		o.Hotstuff2.IgnoreCheckQC = true
		msgReturn := o.Hotstuff2.RestartHotstuff2()
		if msgReturn != nil {
			o.SendMsg(msgReturn)
		}
	}
}

// AddSyncInfo: add sync information to a message
func (o *Orderer) AddSyncInfo(msg *mgmt.NodeMgmtMsg) {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.AddSyncInfo(msg)
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.AddSyncInfo(msg)
	case common.HOTSTUFF_2_PROTOCOL:
		o.Hotstuff2.AddSyncInfo(msg)
	case common.PBFT:
		o.PBFTConsensus.AddSyncInfo(msg)
	}
}

// ClearCurrentRound: clear recieved messages in current round
func (o *Orderer) ClearCurrentRound() {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.ClearCurrentRound()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.ClearCurrentRound()
	}
}

// RefreshLeader: refresh the leader of the view
func (o *Orderer) RefreshLeader() {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.View.RefreshLeader()
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.View.RefreshLeader()
	case common.HOTSTUFF_2_PROTOCOL:
		o.Hotstuff2.View.RefreshLeader()
	case common.PBFT:
		o.PBFTConsensus.View.RefreshLeader()
	}
}

// IsReady: the orderer is ready to start
func (o *Orderer) IsReady() bool {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		return o.BasicHotstuff.View.NodesNum == o.BasicHotstuff.ThresholdSigner.SignNum
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		return o.ChainedHotstuff.View.NodesNum == o.ChainedHotstuff.ThresholdSigner.SignNum
	case common.HOTSTUFF_2_PROTOCOL:
		return o.Hotstuff2.View.NodesNum == o.Hotstuff2.ThresholdSigner.SignNum
	case common.PBFT:
		return true
	default:
		return true
	}
}

// UpdateNodesNum: update the node num
// params:
// - nodeNum: the node number need to update
func (o *Orderer) UpdateNodesNum(nodesNum int) {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.UpdateNodesNum(nodesNum)
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.UpdateNodesNum(nodesNum)
	case common.HOTSTUFF_2_PROTOCOL:
		o.Hotstuff2.UpdateNodesNum(nodesNum)
	case common.PBFT:
		o.PBFTConsensus.UpdateNodesNum(nodesNum)
	}
}

// SyncInfo: sync information from the selected sync-message
// params:
// - msg: the selected sync-message with sync information
// - leader: the leader of this view
func (o *Orderer) SyncInfo(msg *mgmt.NodeMgmtMsg, leader int) {
	switch o.ConsType {
	case common.HOTSTUFF_PROTOCOL_BASIC:
		o.BasicHotstuff.SyncInfo(msg, leader)
	case common.HOTSTUFF_PROTOCOL_CHAINED:
		o.ChainedHotstuff.SyncInfo(msg, leader)
	case common.HOTSTUFF_2_PROTOCOL:
		o.Hotstuff2.SyncInfo(msg, leader)
	case common.PBFT:
		o.PBFTConsensus.SyncInfo(msg, leader)
	}
}
