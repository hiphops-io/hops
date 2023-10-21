package httpserver

import (
	"net/http"
	"strings"

	"github.com/nats-io/nats.go"
)

func Healthcheck(natsConnection *nats.Conn, endpoint string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if (r.Method == "GET" || r.Method == "HEAD") && strings.EqualFold(r.URL.Path, endpoint) {
				w.Header().Set("Content-Type", "text/plain")
				if !natsConnection.IsConnected() {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Not connected to NATS server"))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
				return
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}
