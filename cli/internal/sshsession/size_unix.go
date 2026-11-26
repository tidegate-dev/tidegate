//go:build unix

package sshsession

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/terminal"
)

// followSize tells the shell the new size each time tty's size changes,
// until ctx ends.
func followSize(ctx context.Context, tty *terminal.Terminal, session *ssh.Session) {
	resized := make(chan os.Signal, 1)
	signal.Notify(resized, syscall.SIGWINCH)

	go func() {
		defer signal.Stop(resized)

		for {
			select {
			case <-ctx.Done():
				return
			case <-resized:
				width, height, err := tty.Size()
				if err != nil {
					continue
				}

				_ = session.WindowChange(height, width)
			}
		}
	}()
}
