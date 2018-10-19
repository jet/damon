package win32

import (
	"sync"
	"sync/atomic"
)

// Logger interface for allowing the internal window api calls to make
// log entries in any external logging system
//
// Implementers should implement this simple interface and call SetLogger to enable
// error logging of the win32 package
type Logger interface {
	Error(err error, msg string)
	Logln(v ...interface{})
	Logf(format string, args ...interface{})
}

var globalLoggerLock sync.Mutex
var globalLogger atomic.Value

// SetLogger sets the logger to be used by the win32 api package
// for errors and warnings
func SetLogger(l Logger) {
	globalLoggerLock.Lock()
	defer globalLoggerLock.Unlock()
	globalLogger.Store(logWrapper{logger: l})
}

func init() {
	SetLogger(logWrapper{logger: noopLogger{}})
}

func logger() Logger {
	v := globalLogger.Load()
	return v.(Logger)
}

func Logf(format string, v ...interface{}) {
	logger().Logf(format, v...)
}

func Logln(v ...interface{}) {
	logger().Logln(v...)
}

func LogError(err error, msg string) {
	if err != nil {
		logger().Error(err, msg)
	}
}

type logWrapper struct {
	logger Logger
}

func (n logWrapper) Logf(format string, v ...interface{}) {
	n.logger.Logf(format, v...)
}

func (n logWrapper) Logln(v ...interface{}) {
	n.logger.Logln(v...)
}

func (n logWrapper) Error(err error, msg string) {
	n.logger.Error(err, msg)
}

// noopLogger silently discards logs
type noopLogger struct{}

func (n noopLogger) Logf(format string, v ...interface{}) {

}

func (n noopLogger) Logln(v ...interface{}) {

}

func (n noopLogger) Error(err error, msg string) {

}
