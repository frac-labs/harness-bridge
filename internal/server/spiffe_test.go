package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func ctxWithSPIFFE(t *testing.T, spiffe string) context.Context {
	t.Helper()
	u, err := url.Parse(spiffe)
	if err != nil {
		t.Fatalf("parse %q: %v", spiffe, err)
	}
	cert := &x509.Certificate{URIs: []*url.URL{u}}
	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}},
		},
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

// TestRealTLSInfoSatisfiesAssertion guards against regressing to an anonymous
// interface assertion that real credentials.TLSInfo does not satisfy. The bug
// at v0.2.0 was an assertion to interface{ GetCertificates() []*x509.Certificate },
// which the test fake satisfied but the production type did not — yielding
// "peer auth info not TLS" on every real mTLS call.
func TestRealTLSInfoSatisfiesAssertion(t *testing.T) {
	cert := &x509.Certificate{URIs: mustParseURIs(t, "spiffe://frac-labs/harness/fractury")}
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}},
		},
	})
	id, err := HarnessIDFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id != "fractury" {
		t.Fatalf("id=%q want=fractury", id)
	}
}

func mustParseURIs(t *testing.T, ss ...string) []*url.URL {
	t.Helper()
	out := make([]*url.URL, 0, len(ss))
	for _, s := range ss {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("parse %q: %v", s, err)
		}
		out = append(out, u)
	}
	return out
}
