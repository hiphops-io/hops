package logs

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/rs/zerolog"
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

func UpdateLogContextStr(ctx context.Context, key string, value string) {
	logger := zerolog.Ctx(ctx)
	logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str(key, value)
	})
}
