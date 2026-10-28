// Package dummy implements the DummyService, which checks the path from a
// client through the authority into the database and back.
package dummy

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"

	"github.com/tidegate-dev/tidegate/authority/internal/db/sqlc"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// Service stores texts in the database and reads them back.
type Service struct {
	queries *sqlc.Queries
}

// NewService returns a Service that uses queries.
func NewService(queries *sqlc.Queries) Service {
	return Service{queries: queries}
}

// Write stores the text and returns its ID.
func (s Service) Write(
	ctx context.Context,
	req *connect.Request[authorityv1.WriteRequest],
) (*connect.Response[authorityv1.WriteResponse], error) {
	id, err := s.queries.CreateDummy(ctx, req.Msg.GetText())
	if err != nil {
		return nil, fmt.Errorf("storing the text: %w", err)
	}

	return connect.NewResponse(&authorityv1.WriteResponse{Id: id}), nil
}

// Read returns the text with the given ID.
func (s Service) Read(
	ctx context.Context,
	req *connect.Request[authorityv1.ReadRequest],
) (*connect.Response[authorityv1.ReadResponse], error) {
	text, err := s.queries.GetDummy(ctx, req.Msg.GetId())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no text with ID %d", req.Msg.GetId()))
	}

	if err != nil {
		return nil, fmt.Errorf("reading the text: %w", err)
	}

	return connect.NewResponse(&authorityv1.ReadResponse{Text: text}), nil
}
