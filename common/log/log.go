package log

import (
	"log"
	"os"
	"time"
)

// Logger: responsible for keeping a log
type Logger struct {
	Logger     log.Logger // a logger interface used to perform logging operations
	FPath      string     // the path for storing log files
	TimePoints []int64    // a slice of a point in time that records the specific point in time when logs are recorded
}

// NewLogger: return a new logger
func NewLogger(path string) *Logger {
	return &Logger{
		FPath:      path,
		TimePoints: make([]int64, 0),
	}
}

// LogTime2File: log the current timestamp to file
func (l *Logger) LogTime2File() {
	logFile, err := os.OpenFile(l.FPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	// create a new logger and redirect the output to the log file
	l.Logger = *log.New(logFile, "", log.LstdFlags)

	// record the current time stamp to the log file
	for i := 0; i < len(l.TimePoints); i++ {
		l.Logger.Println(i, "  ", l.TimePoints[i])
	}
}

// AppendPoint: add a point to logger
func (l *Logger) AppendPoint() {
	l.TimePoints = append(l.TimePoints, time.Now().UnixNano()/int64(time.Millisecond))
}
