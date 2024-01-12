package logs

import (
	"net/http"
	"strings"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func AccessLogMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	chain := alice.New()
	chain = chain.Append(hlog.NewHandler(logger))
	chain = chain.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		if strings.HasPrefix(r.URL.Path, "/console") {
			return
		}

		hlog.FromRequest(r).Info().
			Int("status", status).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Str("ip", r.RemoteAddr).
			Str("user-agent", r.UserAgent()).
			Dur("duration", time.Duration(duration)).
			Msg("")
	}))

	return chain.Then
}
