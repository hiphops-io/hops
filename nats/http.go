package nats

import (
	"net/http"
	"strings"
)

func writeNatsConnectionStatus(w http.ResponseWriter, natsClient *Client) {
	w.Header().Set("Content-Type", "text/plain")
	if !natsClient.CheckConnection() {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Not connected to NATS server"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func HealthcheckFunc(natsClient *Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		writeNatsConnectionStatus(w, natsClient)
	}
}

func HealthcheckMiddleware(natsClient *Client, endpoint string) func(http.Handler) http.Handler {
	f := func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if (r.Method == "GET" || r.Method == "HEAD") && strings.EqualFold(r.URL.Path, endpoint) {
				writeNatsConnectionStatus(w, natsClient)
				return
			}
			h.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return f
}
