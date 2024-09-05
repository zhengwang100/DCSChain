package log_test

import (
	"deltachain/common/log"
	"testing"
	"time"
)

func TestLogTime(t *testing.T) {
	testLogger := log.NewLogger("app.log")

	for i := 0; i < 100; i++ {
		testLogger.AppendPoint()
		time.Sleep(10 * time.Microsecond)
	}
	testLogger.LogTime2File()
}
