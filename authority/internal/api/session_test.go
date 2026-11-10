package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
	"github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1/authorityv1connect"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestSessionInterceptorPutsTheSessionIntoTheContext(t *testing.T) {
	t.Parallel()

	// arrange
	session := service.Session{
		ID:     pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		UserID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
	}
	sessions := &fakeSessions{byToken: map[string]service.Session{"abc": session}}
	found := make(chan service.Session, 1)
	client := pingClient(t, NewSessionInterceptor(sessions), connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				inContext, _ := sessionFromContext(ctx)
				found <- inContext

				return next(ctx, req)
			}
		},
	))

	// act
	_, err := client.Ping(t.Context(), pingWithToken("abc"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, session, <-found)
}

func TestSessionInterceptorAddsTheUserAndTheSessionToTheLogger(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	session := service.Session{
		ID:     pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		UserID: pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
	}
	sessions := &fakeSessions{byToken: map[string]service.Session{"abc": session}}
	handler := NewSessionInterceptor(sessions).WrapUnary(
		func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			logging.FromContext(ctx).InfoContext(ctx, "handled")

			return connect.NewResponse(&authorityv1.PingResponse{}), nil
		},
	)
	ctx := logging.NewContext(t.Context(), logging.New(&out, slog.LevelInfo, logging.FormatText))

	// act
	_, err := handler(ctx, pingWithToken("abc"))

	// assert
	require.NoError(t, err)
	assert.Contains(t, out.String(), "user_id="+session.UserID.String())
	assert.Contains(t, out.String(), "session_id="+session.ID.String())
}

func TestSessionInterceptorRefusesACallWithoutAToken(t *testing.T) {
	t.Parallel()

	// arrange
	sessions := &fakeSessions{}
	client := pingClient(t, NewSessionInterceptor(sessions))

	// act
	_, err := client.Ping(t.Context(), connect.NewRequest(&authorityv1.PingRequest{}))

	// assert
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.False(t, sessions.called.Load())
}

func TestSessionInterceptorRefusesAnUnknownTokenWithoutTheReason(t *testing.T) {
	t.Parallel()

	// arrange
	client := pingClient(t, NewSessionInterceptor(&fakeSessions{}))

	// act
	_, err := client.Ping(t.Context(), pingWithToken("unknown"))

	// assert
	var connectErr *connect.Error

	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	assert.Equal(t, "not signed in", connectErr.Message())
}

func TestSessionInterceptorDoesNotReportAFailedLookupAsNotSignedIn(t *testing.T) {
	t.Parallel()

	// arrange
	client := pingClient(t, NewSessionInterceptor(&fakeSessions{err: errors.New("connection refused")}))

	// act
	_, err := client.Ping(t.Context(), pingWithToken("abc"))

	// assert
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}

func TestSessionInterceptorLetsAPublicProcedureThroughWithoutAToken(t *testing.T) {
	t.Parallel()

	// arrange
	sessions := &fakeSessions{}
	client := pingClient(t, NewSessionInterceptor(sessions, authorityv1connect.PingServicePingProcedure))

	// act
	_, err := client.Ping(t.Context(), connect.NewRequest(&authorityv1.PingRequest{}))

	// assert
	require.NoError(t, err)
	assert.False(t, sessions.called.Load())
}

func TestSessionInterceptorRefusesAStreamWithoutAToken(t *testing.T) {
	t.Parallel()

	// arrange
	reached := false
	handler := NewSessionInterceptor(&fakeSessions{}).WrapStreamingHandler(
		func(context.Context, connect.StreamingHandlerConn) error {
			reached = true

			return nil
		},
	)

	// act
	err := handler(t.Context(), streamWithoutToken{})

	// assert
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.False(t, reached)
}

func TestBearerTokenReadsTheAuthorizationHeader(t *testing.T) {
	t.Parallel()

	for name, test := range map[string]struct {
		header string
		token  string
		ok     bool
	}{
		"bearer token":    {"Bearer abc", "abc", true},
		"any letter case": {"bEaReR abc", "abc", true},
		"no header":       {"", "", false},
		"another scheme":  {"Basic YWxpY2U6c2VjcmV0", "", false},
		"no token":        {"Bearer ", "", false},
		"only a scheme":   {"Bearer", "", false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// arrange
			header := http.Header{}
			if test.header != "" {
				header.Set("Authorization", test.header)
			}

			// act
			token, ok := bearerToken(header)

			// assert
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.token, token.Reveal())
		})
	}
}

// fakeSessions knows the sessions in byToken, or fails every lookup with err.
type fakeSessions struct {
	byToken map[string]service.Session
	err     error
	called  atomic.Bool
}

func (f *fakeSessions) Session(_ context.Context, token secret.String) (service.Session, error) {
	f.called.Store(true)

	if f.err != nil {
		return service.Session{}, f.err
	}

	session, ok := f.byToken[token.Reveal()]
	if !ok {
		return service.Session{}, fmt.Errorf("%w: unknown or expired token", service.ErrNotSignedIn)
	}

	return session, nil
}

type streamWithoutToken struct {
	connect.StreamingHandlerConn
}

func (streamWithoutToken) Spec() connect.Spec {
	return connect.Spec{Procedure: "/tidegate.authority.v1.SomeService/Stream"}
}

func (streamWithoutToken) RequestHeader() http.Header {
	return http.Header{}
}

// pingClient returns a client for a PingService behind interceptors.
func pingClient(t *testing.T, interceptors ...connect.Interceptor) authorityv1connect.PingServiceClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(authorityv1connect.NewPingServiceHandler(PingService{}, connect.WithInterceptors(interceptors...)))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return authorityv1connect.NewPingServiceClient(server.Client(), server.URL)
}

func pingWithToken(token string) *connect.Request[authorityv1.PingRequest] {
	request := connect.NewRequest(&authorityv1.PingRequest{})
	request.Header().Set("Authorization", "Bearer "+token)

	return request
}
