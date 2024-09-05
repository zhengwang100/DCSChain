package hstypes

import common "common"

// ChainedQC: qurom certificate in chained hotstuff
type ChainedQC struct {
	QType      StateType        // the type of qurom certificate
	ViewNumber int              // the view number
	HsNodes    [4]common.HsNode // four qurom certificate nodes
	Sign       []byte           // part signature or complete signature
}
