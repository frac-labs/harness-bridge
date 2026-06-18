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
// Accepted SPIFFE URIs (v0.1.0):
//   - spiffe://harness.fractura/frac
//   - spiffe://harness.fractura/fractury
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
		case strings.HasPrefix(s, "spiffe://harness.fractura/frac"):
			return "frac", nil
		case strings.HasPrefix(s, "spiffe://harness.fractura/fractury"):
			return "fractury", nil
		}
	}
	return "", errors.New("peer cert has no recognized SPIFFE id")
}
