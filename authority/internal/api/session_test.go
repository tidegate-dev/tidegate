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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/tidegate-dev/tidegate/authority/internal/model"
	"github.com/tidegate-dev/tidegate/authority/internal/service"
	"github.com/tidegate-dev/tidegate/internal/logging"
	"github.com/tidegate-dev/tidegate/internal/secret"
)

func TestSessionInterceptorPutsTheSessionIntoTheContext(t *testing.T) {
	t.Parallel()

	// arrange
	session := model.Session{
		ID:     model.SessionID{1},
		UserID: model.UserID{2},
	}
	sessions := &fakeSessions{byToken: map[string]model.Session{"abc": session}}
	found := make(chan model.Session, 1)
	client := testClient(t, NewSessionInterceptor(sessions), connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				inContext, _ := sessionFromContext(ctx)
				found <- inContext

				return next(ctx, req)
			}
		},
	))

	// act
	_, err := client.CallUnary(t.Context(), requestWithToken("abc"))

	// assert
	require.NoError(t, err)
	assert.Equal(t, session, <-found)
}

func TestSessionInterceptorAddsTheUserAndTheSessionToTheLogger(t *testing.T) {
	t.Parallel()

	// arrange
	var out bytes.Buffer

	session := model.Session{
		ID:     model.SessionID{1},
		UserID: model.UserID{2},
	}
	sessions := &fakeSessions{byToken: map[string]model.Session{"abc": session}}
	handler := NewSessionInterceptor(sessions).WrapUnary(
		func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
			logging.FromContext(ctx).InfoContext(ctx, "handled")

			return connect.NewResponse(&emptypb.Empty{}), nil
		},
	)
	ctx := logging.NewContext(t.Context(), logging.New(&out, slog.LevelInfo, logging.FormatText))

	// act
	_, err := handler(ctx, requestWithToken("abc"))

	// assert
	require.NoError(t, err)
	assert.Contains(t, out.String(), "user_id="+session.UserID.String())
	assert.Contains(t, out.String(), "session_id="+session.ID.String())
}

func TestSessionInterceptorRefusesACallWithoutAToken(t *testing.T) {
	t.Parallel()

	// arrange
	sessions := &fakeSessions{}
	client := testClient(t, NewSessionInterceptor(sessions))

	// act
	_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))

	// assert
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.False(t, sessions.called.Load())
}

func TestSessionInterceptorRefusesAnUnknownTokenWithoutTheReason(t *testing.T) {
	t.Parallel()

	// arrange
	client := testClient(t, NewSessionInterceptor(&fakeSessions{}))

	// act
	_, err := client.CallUnary(t.Context(), requestWithToken("unknown"))

	// assert
	var connectErr *connect.Error

	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	assert.Equal(t, "not signed in", connectErr.Message())
}

func TestSessionInterceptorDoesNotReportAFailedLookupAsNotSignedIn(t *testing.T) {
	t.Parallel()

	// arrange
	client := testClient(t, NewSessionInterceptor(&fakeSessions{err: errors.New("connection refused")}))

	// act
	_, err := client.CallUnary(t.Context(), requestWithToken("abc"))

	// assert
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}

func TestSessionInterceptorLetsAPublicProcedureThroughWithoutAToken(t *testing.T) {
	t.Parallel()

	// arrange
	sessions := &fakeSessions{}
	client := testClient(t, NewSessionInterceptor(sessions, testProcedure))

	// act
	_, err := client.CallUnary(t.Context(), connect.NewRequest(&emptypb.Empty{}))

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

type fakeSessions struct {
	byToken map[string]model.Session
	err     error
	called  atomic.Bool
}

func (f *fakeSessions) FindByToken(_ context.Context, token secret.String) (model.Session, error) {
	f.called.Store(true)

	if f.err != nil {
		return model.Session{}, f.err
	}

	session, ok := f.byToken[token.Reveal()]
	if !ok {
		return model.Session{}, fmt.Errorf("%w: unknown or expired token", service.ErrNotSignedIn)
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

// testProcedure is a unary procedure that the tests serve themselves, so they
// don't depend on a procedure of the API.
const testProcedure = "/tidegate.authority.v1.TestService/Call"

func testClient(t *testing.T, interceptors ...connect.Interceptor) *connect.Client[emptypb.Empty, emptypb.Empty] {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(testProcedure, connect.NewUnaryHandler(
		testProcedure,
		func(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
			return connect.NewResponse(&emptypb.Empty{}), nil
		},
		connect.WithInterceptors(interceptors...),
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return connect.NewClient[emptypb.Empty, emptypb.Empty](server.Client(), server.URL+testProcedure)
}

func requestWithToken(token string) *connect.Request[emptypb.Empty] {
	request := connect.NewRequest(&emptypb.Empty{})
	request.Header().Set("Authorization", "Bearer "+token)

	return request
}
