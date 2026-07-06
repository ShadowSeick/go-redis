package logging

import (
	"log/slog"
	"os"
)

type Logger interface {
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, err error)
}

type simpleLog struct {
	logger *slog.Logger
}

func (s simpleLog) Info(message string, args ...any) {
	s.logger.Info(message, args...)
}

func (s simpleLog) Warn(message string, args ...any) {
	s.logger.Warn(message, args...)
}

func (s simpleLog) Error(message string, err error) {
	s.logger.Error(message, "err", err)
}

func NewStructuredLogger() Logger {
	return simpleLog{
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
}
