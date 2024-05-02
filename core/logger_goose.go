package core

import (
	"os"
	"strings"

	"github.com/pressly/goose/v3"
)

type _GooseLogger struct {
	libLogger Logger
}

func NewGooseLogger(logger Logger) goose.Logger {
	// just skip this wrapper
	logger.SkipCallers += 1
	return _GooseLogger{logger}
}

func (l _GooseLogger) Fatalf(format string, v ...interface{}) {
	// Remove new line from the end of the format string
	format, _ = strings.CutSuffix(format, "\n")
	l.libLogger.Errorf(format, v...)
	os.Exit(1)
}

func (l _GooseLogger) Printf(format string, v ...interface{}) {
	// Remove new line from the end of the format string
	format, _ = strings.CutSuffix(format, "\n")
	l.libLogger.Infof(format, v...)
}
