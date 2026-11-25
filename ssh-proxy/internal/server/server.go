// Package server starts the ssh-proxy with its settings.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/The127/go-random"
	"golang.org/x/crypto/ssh"

	"github.com/phemera-dev/phemera/internal/systemd"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/authority"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/config"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/proxy"
	"github.com/phemera-dev/phemera/ssh-proxy/internal/ticket"
)

// Run serves SSH clients on the listen address of cfg until ctx ends or
// accepting connections fails. It presents a host key that it generates
// anew, so the key changes with every start.
func Run(ctx context.Context, cfg config.Config) error {
	hostKey, err := proxy.GenerateHostKey(random.NewCryptoRandom())
	if err != nil {
		return err
	}

	notifier, err := systemd.Open()
	if err != nil {
		return err
	}

	defer func() {
		_ = notifier.Close()
	}()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listening for SSH clients: %w", err)
	}

	slog.Warn("the ssh-proxy takes tickets that nobody signed and connects every session to the SSH host of the development environment. " +
		"Only run it for development")

	sshServer := proxy.NewServer(hostKey, ticket.Unsigned{}, authority.DevSSHHost{})

	slog.Info("ssh-proxy listening",
		"address", listener.Addr().String(),
		"host_key_fingerprint", ssh.FingerprintSHA256(hostKey.PublicKey()))

	serving, stopServing := context.WithCancel(ctx)
	defer stopServing()

	served := make(chan error, 1)

	go func() {
		served <- sshServer.Serve(serving, listener)

		stopServing()
	}()

	err = notifier.Ready()
	if err != nil {
		slog.Error("telling systemd that the ssh-proxy is ready", "error", err)
	}

	<-serving.Done()

	err = notifier.Stopping()
	if err != nil {
		slog.Error("telling systemd that the ssh-proxy is stopping", "error", err)
	}

	return <-served
}
