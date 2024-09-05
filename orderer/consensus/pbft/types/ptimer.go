package ptypes

import (
	common "common"
)

// PTimer:a guaranteed active timer
type PTimer struct {
	VCMsgSendFlag bool            // the flag of whether VCMsg is sent
	Timer         *common.MyTimer // the timer
}
