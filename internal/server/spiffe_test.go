package server

import (
	"context"
	"crypto/x509"
	"net/url"
	"testing"

	"google.golang.org/grpc/peer"
)

// fakeTLSInfo satisfies credentials.AuthInfo (via AuthType) AND the anonymous
// interface HarnessIDFromContext type-asserts on (GetCertificates).
type fakeTLSInfo struct{ certs []*x509.Certificate }

func (f fakeTLSInfo) AuthType() string                    { return "tls" }
func (f fakeTLSInfo) GetCertificates() []*x509.Certificate { return f.certs }

func ctxWithSPIFFE(t *testing.T, spiffe string) context.Context {
	t.Helper()
	u, err := url.Parse(spiffe)
	if err != nil {
		t.Fatalf("parse %q: %v", spiffe, err)
	}
	cert := &x509.Certificate{URIs: []*url.URL{u}}
	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: fakeTLSInfo{certs: []*x509.Certificate{cert}},
	})
}

func TestHarnessIDFromContext(t *testing.T) {
	cases := []struct {
		name    string
		spiffe  string
		wantID  string
		wantErr bool
	}{
		{"fractury harness", "spiffe://frac-labs/harness/fractury", "fractury", false},
		{"frac harness", "spiffe://frac-labs/harness/frac", "frac", false},
		{"webhook-router service", "spiffe://frac-labs/service/webhook-router", "service/webhook-router", false},
		{"service with trailing path", "spiffe://frac-labs/service/webhook-router/v1", "service/webhook-router", false},
		{"service missing name", "spiffe://frac-labs/service/", "", true},
		{"unknown SPIFFE id", "spiffe://other/identity", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := ctxWithSPIFFE(t, tc.spiffe)
			got, err := HarnessIDFromContext(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if got != tc.wantID {
				t.Fatalf("id=%q want=%q", got, tc.wantID)
			}
		})
	}
}
