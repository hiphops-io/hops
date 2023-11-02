package logs

import "github.com/rs/zerolog"

type NatsZeroLogger struct {
	zerolog.Logger
}

func NewNatsZeroLogger(logger zerolog.Logger, from string) NatsZeroLogger {
	ctxLogger := logger.With().Str("from", from).Logger()
	return NatsZeroLogger{ctxLogger}
}

func (n *NatsZeroLogger) Debugf(format string, v ...interface{}) {
	n.Debug().Msgf(format, v...)
}

func (n *NatsZeroLogger) Errf(err error, format string, v ...interface{}) {
	n.Error().Err(err).Msgf(format, v...)
}

func (n *NatsZeroLogger) Errorf(format string, v ...interface{}) {
	n.Error().Msgf(format, v...)
}

func (n *NatsZeroLogger) Fatalf(format string, v ...interface{}) {
	// Using n.WithLevel() rather than n.Fatal() as we don't want zerolog
	// to automatically exit the program
	n.WithLevel(zerolog.FatalLevel).Msgf(format, v...)
}

func (n *NatsZeroLogger) Infof(format string, v ...interface{}) {
	n.Info().Msgf(format, v...)
}

func (n *NatsZeroLogger) Noticef(format string, v ...interface{}) {
	n.Info().Msgf(format, v...)
}

func (n *NatsZeroLogger) Tracef(format string, v ...interface{}) {
	n.Trace().Msgf(format, v...)
}

func (n *NatsZeroLogger) Warnf(format string, v ...interface{}) {
	n.Warn().Msgf(format, v...)
}
