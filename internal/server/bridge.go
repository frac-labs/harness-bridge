package server

import (
	"context"

	harnessv1 "github.com/frac-labs/harness-protos/gen/go/frac_labs/harness/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EnqueueTicket is wired by B3/B4 (OpenClaw bridge-client + Hermes integration).
func (s *Server) EnqueueTicket(ctx context.Context, req *harnessv1.EnqueueTicketRequest) (*harnessv1.EnqueueTicketResponse, error) {
	_ = req
	return nil, status.Error(codes.Unimplemented,
		"EnqueueTicket lands in B3/B4 (cdv#11/#12); B2 ships the chassis only")
}

// EmitEvent is wired by B3/B4.
func (s *Server) EmitEvent(ctx context.Context, req *harnessv1.EmitEventRequest) (*harnessv1.EmitEventResponse, error) {
	_ = req
	return nil, status.Error(codes.Unimplemented,
		"EmitEvent lands in B3/B4 (cdv#11/#12); B2 ships the chassis only")
}
