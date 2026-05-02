package app

import "log"

type stdLogger struct{}

func (l *stdLogger) Errorf(format string, args ...any) {
	log.Printf(format, args...)
}
