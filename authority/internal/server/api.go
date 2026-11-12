package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/The127/go-clock"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tidegate-dev/tidegate/authority/internal/api"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/certreload"
	"github.com/tidegate-dev/tidegate/internal/handlererrors"
	"github.com/tidegate-dev/tidegate/internal/requestlog"
)

// requestlog comes first, so it logs the code the client gets after
// handlererrors has changed it.
func interceptors(clk clock.Service, sessions api.SessionLookup) []connect.Interceptor {
	return []connect.Interceptor{
		requestlog.Interceptor(clk),
		handlererrors.Interceptor(),
		api.NewSessionInterceptor(sessions,
			authorityv1connect.LoginServiceLoginProcedure,
			authorityv1connect.PingServicePingProcedure,
		),
		validate.NewInterceptor(),
	}
}

// apiServer serves plain HTTP if certificate is nil.
func apiServer(
	address string,
	certificate *certreload.Certificate,
	pool *pgxpool.Pool,
	logins *service.Logins,
	sessions api.SessionLookup,
) *http.Server {
	options := connect.WithInterceptors(interceptors(clock.NewSystemClock(), sessions)...)

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(api.PingService{}, options))
	mux.Handle(authorityv1connect.NewDummyServiceHandler(api.NewDummyService(service.NewDummies(pool)), options))
	mux.Handle(api.NewLoginServiceHandler(logins, options))

	// gRPC clients need HTTP/2, and the CLI's Connect client uses HTTP/1.1
	// by default.
	var protocols http.Protocols
	protocols.SetHTTP1(true)

	var tlsConfig *tls.Config

	if certificate != nil {
		tlsConfig = &tls.Config{GetCertificate: certificate.Get}

		protocols.SetHTTP2(true)
	} else {
		protocols.SetUnencryptedHTTP2(true)
	}

	return &http.Server{
		Addr:              address,
		Handler:           mux,
		Protocols:         &protocols,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		// Every method is unary with a small request. A streaming method
		// needs its own limit.
		ReadTimeout: 10 * time.Second,
	}
}
