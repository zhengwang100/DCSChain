package core

import (
	"encoding/json"
	hs2types "hotstuff2/types"
)

// StartViewChange: start the view-change protocol when the timer expire
// StartViewChange implement Hotstuff description as follow:
// – send a timeout message ⟨wish, 𝑣 + 1⟩ to the 𝑡 + 1 view leaders in the epoch
// – any one of the 𝑡 + 1 leaders that collects 2𝑡 + 1 ⟨wish, 𝑣 + 1⟩ messages forming a 𝑇𝐶𝑣+1, or obtains 𝑇𝐶𝑣+1, broadcasts the TC to all parties.
func (hs2 *Hotstuff2) StartViewChange() {
	// generate wish message with signature and send it
	wishMsg := &hs2types.H2Msg{
		MType:      hs2types.WISH,
		ViewNumber: hs2.View.ViewNumber + 1,
		SendNode:   hs2.GetNodeName(),
		ReciNode:   hs2.GetNextLeaderName(),
	}
	wishMsgSgin, err := hs2.ThresholdSigner.ThresholdSign(wishMsg.Message2Byte())
	if err == nil {
		wishMsg.ConsSign = wishMsgSgin
	}
	hs2.PM.WishSendFlag = true
	wishMsgJson, err := json.Marshal(wishMsg)
	if err == nil {
		if hs2.ForwardChan != nil {
			hs2.ForwardChan <- wishMsgJson
		}
		if hs2.SendChan != nil {
			hs2.SendSerMsg(wishMsg)
		}
	}
}

// HandleWish: the next leader of view v+1 (the current view is v) handle the wish message collect 2f+1 wish message
func (hs2 *Hotstuff2) HandleWish(msg *hs2types.H2Msg) *hs2types.H2Msg {

	if _, ok := hs2.PM.WishMsgs[msg.ViewNumber]; !ok {
		// if not exists, add it a bew slice belong to this view number
		hs2.PM.WishMsgs[msg.ViewNumber] = []*hs2types.H2Msg{msg}
	} else {
		hs2.PM.WishMsgs[msg.ViewNumber] = append(hs2.PM.WishMsgs[msg.ViewNumber], msg)
	}

	// check the threshold
	if len(hs2.PM.WishMsgs[msg.ViewNumber]) != (hs2.View.NodesNum-1)/3*2+1 {
		return nil
	}

	// recover the signed message and generate the wish signature
	wishMsg := hs2types.H2Msg{
		MType:      hs2types.WISH,
		ViewNumber: msg.ViewNumber,
	}
	wishSign := hs2.CombineSign(hs2.PM.WishMsgs[msg.ViewNumber], wishMsg)

	// generate message with TC and delete handled wish messages of this view for liveness
	TCMSG := hs2types.H2Msg{
		MType:      hs2types.TCMSG,
		ViewNumber: msg.ViewNumber,
		ReciNode:   "Broadcast",
		ConsSign:   wishSign,
	}
	delete(hs2.PM.WishMsgs, msg.ViewNumber)

	// log
	hs2.Logger.Println("[WISH]: r_", hs2.ConsId, "Succeed!")

	return &TCMSG
}
