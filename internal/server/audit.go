package server

import (
	"encoding/json"
	"log/slog"
	"time"
)

// AuditSink emits per-call audit lines. v0.1.0 emits JSON to stdout; the
// Loki push path activates when --loki-url is set (deferred until D2 lands).
type AuditSink struct {
	logger  *slog.Logger
	lokiURL string
}

// AuditEvent is one audit record.
type AuditEvent struct {
	TS            time.Time `json:"ts"`
	Method        string    `json:"method"`
	HarnessID     string    `json:"harness_id"`
	AppName       string    `json:"app_name,omitempty"`
	Repo          string    `json:"repo,omitempty"`
	TokenIDHash   string    `json:"token_id_hash,omitempty"`
	LatencyMillis int64     `json:"latency_ms,omitempty"`
	Err           string    `json:"err,omitempty"`
}

// NewAuditSink constructs an audit sink. lokiURL=="" emits stdout-only.
func NewAuditSink(logger *slog.Logger, lokiURL string) *AuditSink {
	return &AuditSink{logger: logger, lokiURL: lokiURL}
}

// Emit writes one audit event.
func (a *AuditSink) Emit(ev AuditEvent) {
	if ev.TS.IsZero() {
		ev.TS = time.Now().UTC()
	}
	b, _ := json.Marshal(ev)
	// stdout JSON line — picked up by k8s log shipper into Loki once D2 lands
	a.logger.Info("audit", "event", json.RawMessage(b))
	if a.lokiURL != "" {
		// Loki push path: deferred. When D2 lands, POST to
		// {a.lokiURL}/loki/api/v1/push with streams[].labels={job:harness-bridge,...}
		// and values=[[ts_ns, json_line]].
		_ = a.lokiURL
	}
}
