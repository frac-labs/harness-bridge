// Command bridge is the harness-bridge gRPC server.
//
// Implements MintGitHubToken (GH App installation-token mint) + audit logging.
// mTLS is terminated at the bridge: every RPC requires a client cert signed by
// the harness-ca configured via --client-ca. Peer SPIFFE id determines the
// caller's harness identity for audit attribution.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/frac-labs/harness-bridge/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	var (
		listen    = flag.String("listen", ":9443", "gRPC listen address")
		tlsCert   = flag.String("tls-cert", "/run/secrets/tls/tls.crt", "server TLS cert (PEM)")
		tlsKey    = flag.String("tls-key", "/run/secrets/tls/tls.key", "server TLS key (PEM)")
		clientCA  = flag.String("client-ca", "/run/secrets/harness-ca/ca.crt", "trusted client CA bundle (PEM)")
		ssmRegion = flag.String("ssm-region", "us-west-2", "AWS region for SSM-backed GH App key reads")
		keysDir   = flag.String("keys-dir", "/run/secrets/gh-app-keys", "directory containing GH App private keys (ESO-mounted tmpfs)")
		lokiURL   = flag.String("loki-url", "", "Loki push URL; empty = stdout JSON audit only")
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cert, err := tls.LoadX509KeyPair(*tlsCert, *tlsKey)
	if err != nil {
		logger.Error("tls keypair load failed", "err", err)
		os.Exit(1)
	}
	caBytes, err := os.ReadFile(*clientCA)
	if err != nil {
		logger.Error("client CA read failed", "err", err)
		os.Exit(1)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		logger.Error("client CA parse failed")
		os.Exit(1)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS13,
	}

	srv, err := server.New(server.Config{
		Logger:    logger,
		KeysDir:   *keysDir,
		SSMRegion: *ssmRegion,
		LokiURL:   *lokiURL,
	})
	if err != nil {
		logger.Error("server init failed", "err", err)
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		logger.Error("listen failed", "err", err, "addr", *listen)
		os.Exit(1)
	}
	gs := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsCfg)))
	srv.Register(gs)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		logger.Info("shutting down")
		gs.GracefulStop()
	}()

	logger.Info("harness-bridge listening", "addr", *listen, "ssm_region", *ssmRegion, "loki_url", *lokiURL)
	if err := gs.Serve(lis); err != nil {
		logger.Error("serve failed", "err", err)
		os.Exit(1)
	}
}
