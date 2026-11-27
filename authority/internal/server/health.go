package server

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/phemera-dev/phemera/internal/logging"
)

// readinessTimeout is how long /readyz waits for the database. Kubernetes
// gives up on a probe after a second by default.
const readinessTimeout = time.Second

// pinger is the database that /readyz checks. *pgxpool.Pool is one.
type pinger interface {
	Ping(ctx context.Context) error
}

// liveness answers /healthz. It never asks the database, so a database
// outage doesn't make Kubernetes restart every authority.
func liveness(w http.ResponseWriter, _ *http.Request) {
	writeOK(w)
}

// readiness answers /readyz with 200 if the database answers a ping within
// readinessTimeout, and with 503 otherwise. It only logs when its answer
// changes, so a probe every few seconds doesn't fill the log.
type readiness struct {
	database pinger
	// failing starts false, because the authority only serves once the
	// database has answered.
	failing atomic.Bool
}

func (c *readiness) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// A prober that gives up early says nothing about the database, so its
	// request doesn't cut the ping short.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), readinessTimeout)
	defer cancel()

	err := c.database.Ping(ctx)
	if err != nil {
		if !c.failing.Swap(true) {
			logging.FromContext(ctx).WarnContext(ctx, "the authority isn't ready, because the database doesn't answer", "error", err)
		}

		http.Error(w, "the database doesn't answer", http.StatusServiceUnavailable)

		return
	}

	if c.failing.Swap(false) {
		logging.FromContext(ctx).InfoContext(ctx, "the authority is ready again, because the database answers")
	}

	writeOK(w)
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "ok\n")
}
