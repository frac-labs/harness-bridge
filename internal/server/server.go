// Package server hosts the gRPC service implementations for harness-bridge.
package server

import (
	"log/slog"

	"github.com/frac-labs/harness-bridge/internal/github"
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
type Server struct {
	cfg    Config
	gh     *github.Client
	audit  *AuditSink
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

// Register attaches all sub-services to the gRPC server.
// v0.1.0: services are registered as stubs at the protos-defined service names.
// Real wire-level integration arrives once harness-protos goes GA-stable on the
// service descriptors used here (tracked in B3/B4).
func (s *Server) Register(gs *grpc.Server) {
	// NOTE: harness-protos service registration is intentionally deferred for
	// v0.1.0. The image is publishable + chart-deployable; concrete service
	// wiring lands in B3 (Frac bridge-client integration) where the protos
	// surface stabilizes.
	_ = gs
}
