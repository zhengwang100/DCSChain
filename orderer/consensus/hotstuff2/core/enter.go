package core

import (
	"encoding/json"
	"fmt"
	hs2types "hotstuff2/types"
	"strconv"
)

// HandleNewView: the leader of the view v handle new-view message in hotstuff-2 protocol
// HandleNewView is responsible for handling new-view messages while the leader waits for the EnterTimer to expire
// note: the new-view message will be handled by the GenProposal after EnterTimer expired.
func (hs2 *Hotstuff2) HandleNewView(msg *hs2types.H2Msg) *hs2types.H2Msg {
	// hs2.Logger.Println(msg.SendNode, msg.Justify2.QType, hs2.GetNodeName(), hs2.GetLeaderName())
	if hs2.CurPhase != hs2types.NEW_VIEW && hs2.CurPhase != hs2types.NEW_PROPOSE {
		return nil
	}

	// check this message carrying two qurom certification
	if !hs2.CheckQC(&msg.Justify1, hs2types.PROPOSE, msg.Justify1.ViewNumber, hs2.BlkStore.Height) {
		fmt.Println("[ERROR]: HandleNewView CheckQC1", msg.SendNode, msg)
		return nil
	}
	if !hs2.CheckQC(&msg.Justify2, hs2types.PREPARE, msg.Justify2.ViewNumber, msg.Justify2.Height) {
		fmt.Println("[ERROR]: HandleNewView CheckQC2", msg.SendNode, msg)
		return nil
	}

	// check whether message is from this view
	if hs2.View.ViewNumber == msg.ViewNumber+1 {
		hs2.NewViewMsgs = append(hs2.NewViewMsgs, msg)
	}

	// check threshold
	// if len(hs2.NewViewMsgs) != (hs2.View.NodesNum-1)/3*2+1 {
	// 	return nil
	// }

	// // get the highest single and double certificated QC
	// oneIndex, doubleIndex := GetHighestCertIndex(hs2.NewViewMsgs)

	// // update local proposal QC and prepare QC
	// hs2.ProposalQC = hs2.NewViewMsgs[oneIndex].Justify1
	// hs2.PrepareQC = hs2.NewViewMsgs[doubleIndex].Justify2
	// hs2.Logger.Println("[NEW-VIEW]:", hs2.GetNodeName(), "Succeed in view", hs2.View.ViewNumber, "!")
	// return &hs2types.H2Msg{
	// 	MType:    hs2types.NEW_PROPOSE,
	// 	ReciNode: hs2.GetNodeName(),
	// }
	return nil
}

// HandleOptimisticEnter: the node handle the message with double certificate 𝐶𝑣−1(𝐶𝑣−1(𝐵𝑘−1))
// HandleOptimisticEnter implement hotstuff-2 description as follow:
// Leader 𝐿𝑣. If entering view 𝑣 using 𝐶𝑣−1(𝐶𝑣−1(𝐵𝑘−1)), proceeds directly to the propose step.
// Party. If entering view 𝑣 using 𝐶𝑣−1(𝐶𝑣−1(𝐵𝑘−1)), proceeds directly to the vote step.
func (hs2 *Hotstuff2) HandleOptimisticEnter(msg *hs2types.H2Msg) *hs2types.H2Msg {
	hs2.ProposalLock.Lock()
	defer hs2.ProposalLock.Unlock()

	if hs2.View.ViewNumber != msg.ViewNumber || hs2.CurPhase == hs2types.NEW_PROPOSE {
		return nil
	}

	if !hs2.CheckQC(&msg.Justify2, hs2types.PREPARE, hs2.View.ViewNumber, msg.Justify2.Height) {
		fmt.Println("[ERROR]: HandleOptimisticEnter CheckQC")
		return nil
	}

	// set pacemaker OptimisticFlag true to show that the view is entered by the double certification
	// update local consensus state included locked blk and view number
	hs2.PM.OptimisticFlag = true
	hs2.UpdateConsensus()
	hs2.View.NextView()

	// set view timer for the new view, which will be stopped in vote2 phase in normal case
	// when the view timer expired, the node will send the <WISH, v+1> message to the leader of view v+1
	hs2.PM.ViewTimer.Start(func() {
		// hs2.Logger.Println("View Timer expired", hs2.GetNodeName())
		hs2.StartViewChange()
	}, func() {
		// hs2.Logger.Println("View Timer stop")
	})

	// leader will enter the propose phase and generate a new proposal to start a new round consensus
	// replica will update local state and wait for propose message sent by the leader
	if hs2.IsLeader() {
		hs2.CurPhase = hs2types.NEW_PROPOSE
		// hs2.Logger.Println("[Optime]:", hs2.GetNodeName(), hs2.View.ViewNumber, " Succeed!")
		// if !hs2.CurProposal.IsEmpty() {
		// 	return &hs2types.H2Msg{
		// 		MType:    hs2types.NEW_PROPOSE,
		// 		ReciNode: hs2.GetNodeName(),
		// 	}
		// }
		return nil
	} else {
		hs2.CurPhase = hs2types.NEW_VIEW
		hs2.PrepareQC = msg.Justify2
		// hs2.Logger.Println("[Optime replica]:", hs2.GetNodeName(), hs2.View.ViewNumber, " Succeed!")
		return nil
	}
}

// HandleTC: the node handle the message with TC(the set of 2f+1 wish message that leader recieved)
// HandleTC implement hotstuff-2 description as follow:
// Leader 𝐿𝑣. The leader sets a timer 𝑃𝑝𝑐 + Δ, and then proceeds to the propose step.
// Party. The party sends its locked certificate to the leader 𝐿𝑣 and proceeds to the vote step.
func (hs2 *Hotstuff2) HandleTC(msg *hs2types.H2Msg) *hs2types.H2Msg {

	// verify the message's signature
	wishMsg := hs2types.H2Msg{
		MType:      hs2types.WISH,
		ViewNumber: msg.ViewNumber,
	}
	if !hs2.ThresholdSigner.ThresholdSignVerify(wishMsg.Message2Byte(), msg.ConsSign) {
		return nil
	}

	// update the local view to proposed view in message, reset the pacemaker OptimisticFlag
	// hs2.View.ViewNumber = msg.ViewNumber
	for hs2.View.ViewNumber < msg.ViewNumber {
		hs2.View.NextView()
	}

	// leader start ViewTimer and EnterTimer, and EnterTimer will wait 𝑃𝑝𝑐 + Δ.When the EnterTimer is expired the leader proceeds directly to the propose step
	// replica send the locked proposal QC and prepare QC to the leader and start the ViewTimer
	if hs2.IsLeader() {

		if hs2.View.ViewNumber > msg.ViewNumber {
			return nil
		}
		hs2.CurPhase = hs2types.NEW_VIEW
		hs2.NewViewMsgs = append(hs2.NewViewMsgs, &hs2types.H2Msg{
			MType:      hs2types.NEW_VIEW,
			ReciNode:   "r_" + strconv.Itoa(hs2.GetLeaderNum()),
			ViewNumber: hs2.View.ViewNumber - 1,
			Justify1:   hs2.ProposalQC,
			Justify2:   hs2.PrepareQC,
		})

		// leader set the ViewTimer
		hs2.PM.ViewTimer.Start(func() {
			// fmt.Println("HandleTC View Timer expired", hs2.GetNodeName())
			hs2.StartViewChange()
		}, func() {
			// hs2.Logger.Println("HandleTC View Timer stop", hs2.GetNodeName())
		})
		// hs2.Logger.Println("[ENTER]:", hs2.GetNodeName(), "Succeed entering view", hs2.View.ViewNumber)

		// if the leader sets a timer 𝑃𝑝𝑐 + Δ, and then proceeds to the propose step.
		if hs2.CurPhase != hs2types.NEW_PROPOSE {
			hs2.PM.EnterTimer.Start(func() {

				hs2.CurPhase = hs2types.NEW_PROPOSE
				// hs2.Logger.Println("[TIMER-EXPIRE]:", hs2.GetNodeName())
				msg := &hs2types.H2Msg{
					MType:    hs2types.NEW_PROPOSE,
					ReciNode: hs2.GetNodeName(),
				}
				msgJson, err := json.Marshal(msg)
				if err == nil {
					if hs2.ForwardChan != nil {
						hs2.ForwardChan <- msgJson
					}
					if hs2.SendChan != nil {
						hs2.SendSerMsg(msg)
					}
				}

			}, func() {
				// hs2.Logger.Println("[TIMER-STOP]:", hs2.GetNodeName())
			})
			return nil
		}
		return nil
	} else {
		// the replica update local state
		hs2.CurPhase = hs2types.NEW_PROPOSE

		// the replica send locked certificate to the leader and set enter timer
		// it has the same function as the ViewTimer.Start in the previous if clause
		hs2.PM.ViewTimer.Start(func() {
			// hs2.Logger.Println("HandleTC View Timer expired", hs2.GetNodeName(), hs2.GetNextLeaderName())
			hs2.StartViewChange()
		}, func() {
			// hs2.Logger.Println("View Timer stop", hs2.GetNodeName())
		})

		// log
		// hs2.Logger.Println("[ENTER]:", hs2.GetNodeName(), "Succeed entering view", hs2.View.ViewNumber)
		return &hs2types.H2Msg{
			MType:      hs2types.NEW_VIEW,
			ReciNode:   hs2.GetLeaderName(),
			ViewNumber: hs2.View.ViewNumber - 1,
			Justify1:   hs2.ProposalQC,
			Justify2:   hs2.PrepareQC,
		}
	}
}

// GroupByViewNumber: group by view number
func (hs2 *Hotstuff2) GroupByViewNumber(msgs []hs2types.H2Msg) {
	groups := make(map[int][]hs2types.H2Msg)

	for _, msg := range msgs {
		groups[msg.ViewNumber] = append(groups[msg.ViewNumber], msg)
	}
}
