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
//   - spiffe://frac-labs/harness/frac          → "frac"
//   - spiffe://frac-labs/harness/fractury      → "fractury"
//   - spiffe://frac-labs/service/<name>        → "service/<name>"
//
// Services (e.g. webhook-router) are non-harness callers but still need audit
// attribution; the returned id is used verbatim as audit.HarnessID, so the
// "service/" prefix is preserved to keep it distinguishable from harnesses
// with potentially-colliding names.
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
		case strings.HasPrefix(s, "spiffe://frac-labs/service/"):
			name := strings.TrimPrefix(s, "spiffe://frac-labs/service/")
			// Drop anything after the service name (path segments / query)
			// so the audit id stays stable across cert reissues that may
			// append a trailing path component.
			if i := strings.IndexAny(name, "/?"); i >= 0 {
				name = name[:i]
			}
			if name == "" {
				return "", errors.New("service SPIFFE id missing name segment")
			}
			return "service/" + name, nil
		}
	}
	return "", errors.New("peer cert has no recognized SPIFFE id")
}
