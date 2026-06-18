// Package github mints GitHub App installation tokens in-process.
//
// Private keys live on tmpfs (ESO-managed); we read on every mint so a key
// rotation is picked up without restart. The installation-token API call to
// GitHub uses the JWT we just minted; raw private keys never leave the
// cluster network namespace.
package github

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ClientConfig configures the github client.
type ClientConfig struct {
	KeysDir string // directory containing <app>.pem files + app-ids.json
}

// Client mints installation tokens against the GitHub API.
type Client struct {
	cfg    ClientConfig
	http   *http.Client
	mu     sync.Mutex
	cache  map[string]cachedToken
}

type cachedToken struct {
	token       string
	expiresAt   time.Time
	tokenIDHash string
}

// MintRequest selects which GH App and (optionally) which repo to scope the
// installation token to. v0.1.0 returns the org-wide installation token.
type MintRequest struct {
	AppName string
	Repo    string
}

// MintResult is what MintInstallationToken returns.
type MintResult struct {
	Token         string
	ExpiresAt     time.Time
	TokenIDHash   string
	LatencyMillis int64
}

// appIDs maps app-name -> {app_id, installation_id}.
type appIDs map[string]struct {
	AppID          int64 `json:"app_id"`
	InstallationID int64 `json:"installation_id"`
}

// NewClient constructs a Client. Verifies KeysDir is readable but does NOT
// fail if a specific app's key file is missing — that surfaces at mint time.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.KeysDir == "" {
		return nil, errors.New("KeysDir required")
	}
	if _, err := os.Stat(cfg.KeysDir); err != nil {
		// Keys dir not yet mounted is OK at boot; we re-read on each mint.
		_ = err
	}
	return &Client{
		cfg:   cfg,
		http:  &http.Client{Timeout: 10 * time.Second},
		cache: make(map[string]cachedToken),
	}, nil
}

// MintInstallationToken returns a fresh GH App installation token, with
// in-memory caching to ~50m. Token cache is keyed by (app_name).
func (c *Client) MintInstallationToken(ctx context.Context, req MintRequest) (MintResult, error) {
	start := time.Now()
	c.mu.Lock()
	if ent, ok := c.cache[req.AppName]; ok && time.Until(ent.expiresAt) > 10*time.Minute {
		c.mu.Unlock()
		return MintResult{
			Token:         ent.token,
			ExpiresAt:     ent.expiresAt,
			TokenIDHash:   ent.tokenIDHash,
			LatencyMillis: time.Since(start).Milliseconds(),
		}, nil
	}
	c.mu.Unlock()

	ids, err := c.loadAppIDs()
	if err != nil {
		return MintResult{}, fmt.Errorf("load app-ids: %w", err)
	}
	app, ok := ids[req.AppName]
	if !ok {
		return MintResult{}, fmt.Errorf("unknown app: %s", req.AppName)
	}

	keyPath := filepath.Join(c.cfg.KeysDir, req.AppName+".pem")
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return MintResult{}, fmt.Errorf("read key %s: %w", keyPath, err)
	}
	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return MintResult{}, fmt.Errorf("parse key %s: %w", keyPath, err)
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": app.AppID,
	})
	signed, err := tok.SignedString(signKey)
	if err != nil {
		return MintResult{}, fmt.Errorf("sign jwt: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", app.InstallationID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(""))
	if err != nil {
		return MintResult{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+signed)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return MintResult{}, fmt.Errorf("installation token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return MintResult{}, fmt.Errorf("installation token http=%d body=%s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return MintResult{}, fmt.Errorf("parse token response: %w", err)
	}
	h := sha256.Sum256([]byte(parsed.Token))
	tokIDHash := hex.EncodeToString(h[:8])

	c.mu.Lock()
	c.cache[req.AppName] = cachedToken{token: parsed.Token, expiresAt: parsed.ExpiresAt, tokenIDHash: tokIDHash}
	c.mu.Unlock()

	return MintResult{
		Token:         parsed.Token,
		ExpiresAt:     parsed.ExpiresAt,
		TokenIDHash:   tokIDHash,
		LatencyMillis: time.Since(start).Milliseconds(),
	}, nil
}

func (c *Client) loadAppIDs() (appIDs, error) {
	p := filepath.Join(c.cfg.KeysDir, "app-ids.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var ids appIDs
	if err := json.Unmarshal(b, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}
