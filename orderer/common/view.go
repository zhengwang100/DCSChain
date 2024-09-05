package common

import "strconv"

// View: the view of consensus
type View struct {
	ViewNumber int // the consensus at which view
	Leader     int // the unique identity in consensus of the node
	NodesNum   int // the number of nodes participating in the consensus
}

// NextView: go to the next view and update the view number and leader
func (v *View) NextView() {
	v.Leader = (v.Leader + 1) % v.NodesNum
	v.ViewNumber += 1
}

// NextLeader: get the next leader number of next view
func (v *View) NextLeader() {
	v.Leader = (v.Leader + 1) % v.NodesNum
}

// RefreshLeade: refresh the leader of the view
// nodesnum changes because the current view does not start and the original leader exits
func (v *View) RefreshLeader() {
	v.Leader = v.Leader % v.NodesNum
}

// UpdateView: go to the specified view and update the view number and leader
// params:
// - viewNumber: the view number of the specified view
// - leader: the leader number of the specified view
func (v *View) UpdateView(viewNumber int, leader int) {
	v.ViewNumber = viewNumber
	v.Leader = leader
}

// UpdateView: update the nodes number
// params:
// - newNodesNum: the specified new nodes number
func (v *View) UpdateNodesNum(newNodesNum int) {
	v.NodesNum = newNodesNum
}

// LeaderName: get the leader name of the view
// return:
// - the leader name of the view
func (v *View) LeaderName() string {
	return "r_" + strconv.Itoa(v.Leader)
}

// LeaderName: get the leader name of the next view
// return:
// - the leader name of the next view
func (v *View) NextLeaderName() string {
	return "r_" + strconv.Itoa((v.Leader+1)%v.NodesNum)
}

// LeaderName: get the leader name of the last view
// return:
// - the leader name of the last view
func (v *View) LastLeaderName() string {
	return "r_" + strconv.Itoa((v.Leader+v.NodesNum-1)%v.NodesNum)
}
