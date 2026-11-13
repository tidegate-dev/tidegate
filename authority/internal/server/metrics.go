package server

import (
	"net/http"
	"time"
)

func metricsServer(address string, handler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", handler)

	return &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// Longer than Prometheus's default scrape interval of one minute,
		// so a scraper can reuse its connection.
		IdleTimeout: 90 * time.Second,
	}
}
