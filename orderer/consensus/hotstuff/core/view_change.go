package core

import (
	"encoding/json"
	hstypes "hotstuff/types"
)

// StartViewChange: start the view-change protocol when the timer expire
// StartViewChange implement Hotstuff description as follow:
// send Msg(new-view, ⊥, prepareQC) to leader(curView + 1)
func (bhs *BCHotstuff) StartViewChange() {
	// update the view number to expect to enter
	bhs.View.NextView()

	// generate new-view message
	newViewMsg := hstypes.Msg{
		MType:      hstypes.NEW_VIEW,
		Justify:    bhs.PrepareQC,
		ViewNumber: bhs.View.ViewNumber,
		SendNode:   bhs.GetNodeName(),
		ReciNode:   bhs.GetLeaderName(),
	}

	sign, err := bhs.ThresholdSigner.ThresholdSign(newViewMsg.Message2Byte())

	if err == nil {
		newViewMsg.PartialSig = sign
	} else {
		bhs.Logger.Println("[ERROR]", bhs.GetNodeName(), err)
	}

	// send the message
	newViewMsgJson, err := json.Marshal(newViewMsg)
	if err == nil {
		if bhs.ForwardChan != nil {
			bhs.ViewChangeSendFlag = true
			bhs.ForwardChan <- newViewMsgJson
		}
		if bhs.SendChan != nil {
			bhs.SendSerMsg(&newViewMsg)
			bhs.CurRoundMsg = append(bhs.CurRoundMsg, &newViewMsg)
		}
	} else {
		bhs.Logger.Println("[ERROR]", bhs.GetNodeName(), err)
	}

	// log
	bhs.Logger.Println("[VIEW_CHANGE]:", bhs.GetNodeName(), "ViewNumber:", bhs.View.ViewNumber)
	bhs.CurPhase = hstypes.NEW_VIEW

	// set the view timer to ensure to enter the new view correctly for liveness
	bhs.ViewTimer.Start(func() {
		// bhs.Logger.Println("StartViewChange View Timer stop", bhs.GetNodeName())
		bhs.StartViewChange()
	}, func() {
		// bhs.Logger.Println("StartViewChange View Timer stop", bhs.GetNodeName())
	})
}
