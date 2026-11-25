package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/The127/go-clock"

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

// externalAPIServer serves plain HTTP if certificate is nil.
func externalAPIServer(
	address string,
	certificate *certreload.Certificate,
	clk clock.Service,
	dummies *service.Dummies,
	logins *service.Logins,
	projects *service.Projects,
	roles *service.Roles,
	assets *service.Assets,
	bindings *service.Bindings,
	tickets *service.Tickets,
	sessions api.SessionLookup,
) *http.Server {
	options := connect.WithInterceptors(interceptors(clk, sessions)...)

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(api.PingService{}, options))
	mux.Handle(authorityv1connect.NewDummyServiceHandler(api.NewDummyService(dummies), options))
	mux.Handle(api.NewLoginServiceHandler(logins, options))
	mux.Handle(authorityv1connect.NewProjectServiceHandler(api.NewProjectService(projects), options))
	mux.Handle(authorityv1connect.NewRoleServiceHandler(api.NewRoleService(roles), options))
	mux.Handle(authorityv1connect.NewAssetServiceHandler(api.NewAssetService(assets), options))
	mux.Handle(authorityv1connect.NewBindingServiceHandler(api.NewBindingService(bindings), options))
	mux.Handle(authorityv1connect.NewTicketServiceHandler(api.NewTicketService(tickets), options))

	var tlsConfig *tls.Config
	if certificate != nil {
		tlsConfig = &tls.Config{GetCertificate: certificate.Get}
	}

	return newServer(address, tlsConfig, mux)
}

func internalAPIServer(address string, certificate *certreload.Certificate) *http.Server {
	return newServer(address, &tls.Config{GetCertificate: certificate.Get}, http.NewServeMux())
}

func newServer(address string, tlsConfig *tls.Config, handler http.Handler) *http.Server {
	// gRPC clients need HTTP/2, and Connect clients use HTTP/1.1 by default.
	var protocols http.Protocols
	protocols.SetHTTP1(true)

	if tlsConfig != nil {
		protocols.SetHTTP2(true)
	} else {
		protocols.SetUnencryptedHTTP2(true)
	}

	return &http.Server{
		Addr:              address,
		Handler:           handler,
		Protocols:         &protocols,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 5 * time.Second,
		// Every method is unary with a small request. A streaming method
		// needs its own limit.
		ReadTimeout: 10 * time.Second,
	}
}
