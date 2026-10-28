// Package systemd tells systemd when a service is ready and when it stops,
// for units with Type=notify.
package systemd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mdlayher/sdnotify"
)

// Notifier sends a service's state to systemd. Outside systemd it sends
// nothing.
type Notifier struct {
	notifier *sdnotify.Notifier
}

// Open connects to the socket that systemd names in NOTIFY_SOCKET. If the
// variable isn't set, the service doesn't run under systemd, and the returned
// Notifier sends nothing.
func Open() (*Notifier, error) {
	notifier, err := sdnotify.New()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("connecting to systemd: %w", err)
	}

	return &Notifier{notifier: notifier}, nil
}

// Ready tells systemd that the service accepts connections.
func (n *Notifier) Ready() error {
	return n.notifier.Notify(sdnotify.Ready)
}

// Stopping tells systemd that the service has begun to shut down.
func (n *Notifier) Stopping() error {
	return n.notifier.Notify(sdnotify.Stopping)
}

// Close closes the connection to systemd.
func (n *Notifier) Close() error {
	return n.notifier.Close()
}
