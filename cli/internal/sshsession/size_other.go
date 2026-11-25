//go:build !unix

package sshsession

import (
	"context"

	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/terminal"
)

// Only Unix has a signal for a change of the terminal's size.
func followSize(context.Context, *terminal.Terminal, *ssh.Session) {}
