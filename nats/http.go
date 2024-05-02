package nats

import "net/http"

func HealthChecker(natsClient *Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if !natsClient.CheckConnection() {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Not connected to NATS cluster"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
