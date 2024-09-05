package common

import (
	"time"
)

// MyTimer: a repackaged timer used to trigger ViewChange when a consensus timeout occurs
type MyTimer struct {
	duration     time.Duration // timeout period of the timer
	timer        *time.Timer   // timer in the time library
	stopChan     chan bool     // the channel in the timer that receives the stop signal
	IsStopped    bool          // indicate whether the timer is running
	ExpireAction func()        // a function that runs after the timer expires
	StopAction   func()        // a function that runs after the timer stops
}

// NewTimer: generate a new timer
// params:
// - duration: timeout period of the timer
// return
// - a new timer
func NewTimer(duration time.Duration) *MyTimer {
	return &MyTimer{
		duration:  duration,
		stopChan:  make(chan bool),
		IsStopped: true,
	}
}

// Start: start the timer
// params
// - fExpire: 	function that need to be executed after the timer expires
// - fStop: 	function that need to be executed after the timer is stopped
func (t *MyTimer) Start(fExpire func(), fStop func()) {
	t.timer = time.NewTimer(t.duration)

	if !t.IsStopped {
		t.stopChan <- true
	} else {
		t.IsStopped = false
	}

	t.ExpireAction = fExpire
	t.StopAction = fStop
	go func() {
		select {
		case <-time.After(t.duration):
			// execute the instructions you want here
			t.IsStopped = true
			t.ExpireAction()
			t.duration = t.duration * 2
			return
		case <-t.stopChan:
			t.timer.Stop()
			t.StopAction()
			t.duration = 5 * time.Second
			return
		}
	}()
}

// Start: stop the timer
func (t *MyTimer) Stop() {
	if !t.IsStopped {
		t.stopChan <- true
		t.IsStopped = true
	}
}

// ReSet: start the timer but there is no need to update timeout operations and stop stops
func (t *MyTimer) ReSet() {
	t.Stop()
	t.Start(t.ExpireAction, t.StopAction)
}

// Duration: return timeout period of the timer
// return:
// - the timeout period of the timer
func (t *MyTimer) Duration() time.Duration {
	return t.duration
}
