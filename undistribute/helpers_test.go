package undistribute

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func initTestLogger() zerolog.Logger {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return log.Logger
}
