# harness-bridge

In-cluster gRPC service that mints short-lived GitHub App installation tokens
for harness clients (Frac, Fractury) over mTLS. Audits every call. Private
GitHub App keys never leave the cluster network.

Tracks [`frac-labs/clawdiovascular#10`](https://github.com/frac-labs/clawdiovascular/issues/10).

## Status

v0.1.0 — scaffold. Image publishable to `ghcr.io/frac-labs/harness-bridge:v0.1.0`.
Chart + Argo Application land in PR-B (`frac-labs/clawdiovascular` `kit/platform/harness-bridge/`).

Real gRPC service-descriptor wiring (Bridge/Secrets) lands in B3 (Frac
bridge-client) once `harness-protos` finalizes the service surface.

## Build

```bash
go build ./...
docker build -t harness-bridge:dev .
```

## Run

```bash
bridge \
  --listen :9443 \
  --tls-cert /run/secrets/tls/tls.crt \
  --tls-key /run/secrets/tls/tls.key \
  --client-ca /run/secrets/harness-ca/ca.crt \
  --keys-dir /run/secrets/gh-app-keys \
  --ssm-region us-west-2
```

## License

MIT — see `LICENSE`.
