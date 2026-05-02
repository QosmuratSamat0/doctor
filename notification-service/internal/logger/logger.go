package logger

import "log"

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type simpleLogger struct {
	*log.Logger
}

func NewLogger() Logger {
	return &simpleLogger{
		Logger: log.Default(),
	}
}

func (l *simpleLogger) Infof(format string, args ...interface{}) {
	l.Printf("[INFO] "+format, args...)
}

func (l *simpleLogger) Errorf(format string, args ...interface{}) {
	l.Printf("[ERROR] "+format, args...)
}
