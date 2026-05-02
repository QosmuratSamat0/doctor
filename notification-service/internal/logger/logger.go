package logger

import (
	"log"
	"os"
)

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type simpleLogger struct {
	*log.Logger
}

func NewLogger() Logger {
	return &simpleLogger{
		Logger: log.New(os.Stdout, "", 0),
	}
}

func (l *simpleLogger) Infof(format string, args ...interface{}) {
	l.Printf(format, args...)
}

func (l *simpleLogger) Errorf(format string, args ...interface{}) {
	l.Printf(format, args...)
}
