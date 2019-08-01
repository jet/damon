package container

import "io"

type Logger interface {
	Logln(v ...interface{})
	Error(err error, msg string)
}

type logWrapper struct {
	Logger Logger
}

func (l logWrapper) Logln(v ...interface{}) {
	if l.Logger == nil {
		return
	}
	l.Logln(v...)
}
func (l logWrapper) Error(err error, msg string) {
	if l.Logger == nil {
		return
	}
	l.Error(err, msg)
}

func (l logWrapper) CloseLogError(c io.Closer, msg string) {
	if err := c.Close(); err != nil {
		l.Error(err, msg)
	}
}
