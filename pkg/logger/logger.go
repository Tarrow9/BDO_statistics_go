package logger

import "log"

type Logger interface {
	Infof(format string, v ...any)
	Errorf(format string, v ...any)
}

type stdLogger struct{}

func New() Logger { return &stdLogger{} }

func (l *stdLogger) Infof(format string, v ...any)  { log.Printf("[INFO] "+format, v...) }
func (l *stdLogger) Errorf(format string, v ...any) { log.Printf("[ERROR] "+format, v...) }
