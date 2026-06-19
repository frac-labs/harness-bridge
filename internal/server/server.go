// Package server hosts the gRPC service implementations for harness-bridge.
package server

import (
	"log/slog"

	"github.com/frac-labs/harness-bridge/internal/github"
	harnessv1 "github.com/frac-labs/harness-protos/gen/go/frac_labs/harness/v1"
	"google.golang.org/grpc"
)

// Config wires server-level dependencies from main.
type Config struct {
	Logger    *slog.Logger
	KeysDir   string
	SSMRegion string
	LokiURL   string
}

// Server is the gRPC service container.
//
// Embeds the v1 Unimplemented*Server types so that future RPCs added to the
// harness-protos surface compile cleanly without an immediate code change here.
type Server struct {
	harnessv1.UnimplementedSecretsServer
	harnessv1.UnimplementedBridgeServer

	cfg   Config
	gh    *github.Client
	audit *AuditSink
}

// New constructs the server.
func New(cfg Config) (*Server, error) {
	ghc, err := github.NewClient(github.ClientConfig{KeysDir: cfg.KeysDir})
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:   cfg,
		gh:    ghc,
		audit: NewAuditSink(cfg.Logger, cfg.LokiURL),
	}, nil
}

// Register attaches Secrets + Bridge services to the gRPC server at the
// harness-protos v1 service names. MintGitHubToken is the only RPC with a real
// implementation in this milestone (B2); GetSecret / EnqueueTicket / EmitEvent
// return codes.Unimplemented and are wired up by B3 / B4.
func (s *Server) Register(gs *grpc.Server) {
	harnessv1.RegisterSecretsServer(gs, s)
	harnessv1.RegisterBridgeServer(gs, s)
}
