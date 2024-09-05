package common_test

import (
	common "common"
	"fmt"
	"testing"
	"time"
)

// TestTimer: test the timer
func TestTimer(t *testing.T) {
	timer := common.NewTimer(2 * time.Second)
	timer.Start(func() {
		fmt.Println("timer expire")
	}, func() {
		fmt.Println("timer stop")
	})
	// 等待一段时间后停止定时器
	time.Sleep(1 * time.Second)

	timer.Stop()
	fmt.Println(timer.IsStopped)
	timer.Stop()
	fmt.Println(timer.IsStopped)

	timer.Stop()

}
