package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"github.com/tidegate-dev/tidegate/authority/internal/service"
	authorityv1 "github.com/tidegate-dev/tidegate/gen/tidegate/authority/v1"
)

// DummyService stores texts and reads them back, to check the path from a
// client through the authority into the database.
type DummyService struct {
	dummies *service.Dummies
}

// NewDummyService returns a DummyService that uses dummies.
func NewDummyService(dummies *service.Dummies) *DummyService {
	return &DummyService{dummies: dummies}
}

// Write stores the text and returns its ID.
func (s *DummyService) Write(
	ctx context.Context,
	req *connect.Request[authorityv1.WriteRequest],
) (*connect.Response[authorityv1.WriteResponse], error) {
	id, err := s.dummies.Write(ctx, req.Msg.GetText())
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&authorityv1.WriteResponse{Id: id}), nil
}

// Read returns the text with the given ID.
func (s *DummyService) Read(
	ctx context.Context,
	req *connect.Request[authorityv1.ReadRequest],
) (*connect.Response[authorityv1.ReadResponse], error) {
	text, err := s.dummies.Read(ctx, req.Msg.GetId())
	if errors.Is(err, service.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no text with ID %d", req.Msg.GetId()))
	}

	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&authorityv1.ReadResponse{Text: text}), nil
}
