package rest

import (
	"io"
	"strings"

	"github.com/southernlabs-io/go-fw/log"
)

type _GinWriter struct {
	write func(nsg string)
}

func NewDefaultGinWriter(logger log.Logger) io.Writer {
	// As of gin v1.9, the stack is not helpful until the 4th caller
	logger.SkipCallers += 4
	return &_GinWriter{
		write: func(msg string) {
			// Remove new line from the end of the msg string
			msg, _ = strings.CutSuffix(msg, "\n")
			logger.Info(msg)
		},
	}
}

func NewDefaultErrorGinWriter(logger log.Logger) io.Writer {
	logger.SkipCallers += 3
	return &_GinWriter{
		write: func(msg string) {
			// Remove new line from the end of the msg string
			msg, _ = strings.CutSuffix(msg, "\n")
			logger.Error(msg)
		},
	}
}

// Write interface implementation for gin-framework
func (w _GinWriter) Write(p []byte) (n int, err error) {
	w.write(string(p))
	return len(p), nil
}
