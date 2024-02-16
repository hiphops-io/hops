package logs

import (
	"io"
	"log"
	"os"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

type LevelWriter struct {
	io.Writer
	ErrOut io.Writer
}

func (l *LevelWriter) WriteLevel(level zerolog.Level, txt []byte) (int, error) {
	if level > zerolog.InfoLevel {
		return l.ErrOut.Write(txt)
	}

	return l.Writer.Write(txt)
}

func InitLogger(debug bool) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	var logWriter io.Writer
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		logWriter = debugWriter()
	} else {
		logWriter = levelWriter()
	}

	logger := zerolog.New(logWriter).With().Timestamp().Logger()
	log.SetFlags(0)
	log.SetOutput(logger)

	return logger
}

func debugWriter() io.Writer {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	return zerolog.ConsoleWriter{Out: os.Stdout}
}

func levelWriter() io.Writer {
	return &LevelWriter{os.Stdout, os.Stderr}
}

func NoOpLogger() zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return zlog.Logger
}
