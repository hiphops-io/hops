package logs

import (
	"io"
	"log"
	"os"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func InitLogger(debug bool) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMicro

	var logWriter io.Writer
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logWriter = zerolog.ConsoleWriter{Out: os.Stdout}
	} else {
		logWriter = os.Stdout
	}

	logger := zerolog.New(logWriter).With().Timestamp().Logger()
	log.SetFlags(0)
	log.SetOutput(logger)

	return logger
}

func NoOpLogger() zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return zlog.Logger
}
