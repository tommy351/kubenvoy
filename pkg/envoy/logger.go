package envoy

import (
	"github.com/rs/zerolog"
)

type Logger struct {
	logger *zerolog.Logger
}

func NewLogger(logger *zerolog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Debug().Msgf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error().Msgf(format, args...)
}
