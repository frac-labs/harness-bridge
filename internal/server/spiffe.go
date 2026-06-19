package server

import (
	"context"
	"crypto/x509"
	"errors"
	"strings"

	"google.golang.org/grpc/peer"
)

// HarnessIDFromContext extracts the SPIFFE-id / SAN-URI from the peer cert
// and maps it to a known harness identity. Used for audit attribution.
//
// Accepted SPIFFE URIs (canonical, matching kit/platform/harness-ca/* Certificates):
//   - spiffe://frac-labs/harness/frac
//   - spiffe://frac-labs/harness/fractury
func HarnessIDFromContext(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", errors.New("no peer in context")
	}
	tlsInfo, ok := p.AuthInfo.(interface{ GetCertificates() []*x509.Certificate })
	if !ok {
		return "", errors.New("peer auth info not TLS")
	}
	certs := tlsInfo.GetCertificates()
	if len(certs) == 0 {
		return "", errors.New("no client cert presented")
	}
	for _, uri := range certs[0].URIs {
		s := uri.String()
		switch {
		case strings.HasPrefix(s, "spiffe://frac-labs/harness/fractury"):
			return "fractury", nil
		case strings.HasPrefix(s, "spiffe://frac-labs/harness/frac"):
			return "frac", nil
		}
	}
	return "", errors.New("peer cert has no recognized SPIFFE id")
}
