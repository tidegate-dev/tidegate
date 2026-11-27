package server

import (
	"net/http"
	"time"
)

// metricsServer serves the metrics and the health checks. /readyz pings
// database. None of them needs a login.
func metricsServer(address string, handler http.Handler, database pinger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", handler)
	mux.HandleFunc("GET /healthz", liveness)
	mux.Handle("GET /readyz", &readiness{database: database})

	return &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// Longer than Prometheus's default scrape interval of one minute,
		// so a scraper can reuse its connection.
		IdleTimeout: 90 * time.Second,
	}
}
