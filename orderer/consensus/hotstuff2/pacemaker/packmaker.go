package pacemaker

import (
	common "common"
	hs2types "hotstuff2/types"
)

// Pacemaker: the struct for liveness
type Pacemaker struct {
	WishSendFlag   bool                      // the flag to send wish message
	OptimisticFlag bool                      // if true, the node recieve the last view double certificated block
	EnterTimer     common.MyTimer            // set timer for the enter phase P_pc+Δ
	ViewTimer      common.MyTimer            // set timer for a new view
	WishMsgs       map[int][]*hs2types.H2Msg // the wish messages
}
