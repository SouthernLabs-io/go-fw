package core

import "log/slog"

func NewSlogLogger(logger Logger) *slog.Logger {
	return slog.New(logger.h)
}
