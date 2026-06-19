package server

import (
	"context"
	"errors"
	"time"

	"github.com/frac-labs/harness-bridge/internal/github"
	harnessv1 "github.com/frac-labs/harness-protos/gen/go/frac_labs/harness/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MintGitHubToken implements harnessv1.SecretsServer. The harness identity
// claimed in the request is cross-checked against the SPIFFE id on the peer
// cert so a client cannot mint a token as another harness.
func (s *Server) MintGitHubToken(ctx context.Context, req *harnessv1.MintGitHubTokenRequest) (*harnessv1.MintGitHubTokenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if req.GetAppName() == "" {
		return nil, status.Error(codes.InvalidArgument, "app_name required")
	}

	attestedID, err := HarnessIDFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "spiffe id: %v", err)
	}
	if req.GetHarnessId() != "" && req.GetHarnessId() != attestedID {
		return nil, status.Errorf(codes.PermissionDenied,
			"harness_id mismatch: request=%q peer-cert=%q", req.GetHarnessId(), attestedID)
	}

	repo := ""
	if rs := req.GetRepositories(); len(rs) > 0 {
		repo = rs[0]
	}

	res, err := s.gh.MintInstallationToken(ctx, github.MintRequest{
		AppName: req.GetAppName(),
		Repo:    repo,
	})
	if err != nil {
		s.audit.Emit(AuditEvent{
			Method:    "MintGitHubToken",
			HarnessID: attestedID,
			AppName:   req.GetAppName(),
			Repo:      repo,
			Err:       err.Error(),
		})
		return nil, status.Errorf(codes.Internal, "mint: %v", err)
	}
	s.audit.Emit(AuditEvent{
		Method:        "MintGitHubToken",
		HarnessID:     attestedID,
		AppName:       req.GetAppName(),
		Repo:          repo,
		TokenIDHash:   res.TokenIDHash,
		LatencyMillis: res.LatencyMillis,
	})

	return &harnessv1.MintGitHubTokenResponse{
		Token:     res.Token,
		ExpiresAt: timestamppb.New(res.ExpiresAt),
	}, nil
}

// GetSecret is wired in B3 (cross-harness Secrets retrieval via SSM / k8s).
// At B2 we expose the gRPC entry point so clients can probe the service
// surface, but the call returns Unimplemented with a B3 reference.
func (s *Server) GetSecret(ctx context.Context, req *harnessv1.GetSecretRequest) (*harnessv1.GetSecretResponse, error) {
	_ = req
	return nil, status.Error(codes.Unimplemented,
		"GetSecret lands in B3 (cdv#11); use MintGitHubToken at B2")
}

// inProcessMintGitHubToken keeps the historical in-process API exported for
// unit tests and any non-gRPC callers; the gRPC handler above delegates to
// the github.Client directly to avoid the double-wrap.
//
// nolint:unused // kept for back-compat with smoke harness
func (s *Server) inProcessMintGitHubToken(ctx context.Context, harnessID, appName, repo string) (string, time.Time, error) {
	if appName == "" {
		return "", time.Time{}, errors.New("app_name required")
	}
	res, err := s.gh.MintInstallationToken(ctx, github.MintRequest{AppName: appName, Repo: repo})
	if err != nil {
		return "", time.Time{}, err
	}
	_ = harnessID
	return res.Token, res.ExpiresAt, nil
}
