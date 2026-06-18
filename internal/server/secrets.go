package server

import (
	"context"
	"errors"
	"time"

	"github.com/frac-labs/harness-bridge/internal/github"
)

// MintGitHubToken is the canonical Secrets-service method exercised by the
// done-when test on issue cdv#10. The signature here is the in-process API
// the gRPC handler will adapt to once the protos descriptor stabilizes.
func (s *Server) MintGitHubToken(ctx context.Context, harnessID, appName, repo string) (token string, expiresAt time.Time, err error) {
	if appName == "" {
		return "", time.Time{}, errors.New("app_name required")
	}
	res, err := s.gh.MintInstallationToken(ctx, github.MintRequest{
		AppName: appName,
		Repo:    repo,
	})
	if err != nil {
		s.audit.Emit(AuditEvent{
			Method:    "MintGitHubToken",
			HarnessID: harnessID,
			AppName:   appName,
			Repo:      repo,
			Err:       err.Error(),
		})
		return "", time.Time{}, err
	}
	s.audit.Emit(AuditEvent{
		Method:        "MintGitHubToken",
		HarnessID:     harnessID,
		AppName:       appName,
		Repo:          repo,
		TokenIDHash:   res.TokenIDHash,
		LatencyMillis: res.LatencyMillis,
	})
	return res.Token, res.ExpiresAt, nil
}
