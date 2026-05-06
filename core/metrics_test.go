package core

import (
	"strings"
	"testing"
)

func TestMetricsSnapshotAndPrometheusOutput(t *testing.T) {
	metrics := NewMetrics()
	metrics.IncAcceptedConnection()
	metrics.IncActiveConnection()
	metrics.IncRequest()
	metrics.IncWAFBlock()
	metrics.IncParseReject()
	metrics.IncQueueReject()
	metrics.IncRateLimited()
	metrics.RecordResponse(200)
	metrics.RecordResponse(403)
	metrics.RecordResponse(503)

	snapshot := metrics.Snapshot()
	if snapshot.AcceptedConnections != 1 {
		t.Fatalf("expected accepted connection count, got %d", snapshot.AcceptedConnections)
	}
	if snapshot.ActiveConnections != 1 {
		t.Fatalf("expected active connection count, got %d", snapshot.ActiveConnections)
	}
	if snapshot.TotalRequests != 1 {
		t.Fatalf("expected request count, got %d", snapshot.TotalRequests)
	}
	if snapshot.WAFBlocks != 1 {
		t.Fatalf("expected waf block count, got %d", snapshot.WAFBlocks)
	}
	if snapshot.ParseRejects != 1 {
		t.Fatalf("expected parse reject count, got %d", snapshot.ParseRejects)
	}
	if snapshot.QueueRejects != 1 {
		t.Fatalf("expected queue reject count, got %d", snapshot.QueueRejects)
	}
	if snapshot.RateLimited != 1 {
		t.Fatalf("expected rate limited count, got %d", snapshot.RateLimited)
	}
	if snapshot.Responses2xx != 1 || snapshot.Responses4xx != 1 || snapshot.Responses5xx != 1 {
		t.Fatalf("unexpected response class counts: %+v", snapshot)
	}

	output := metrics.Prometheus()
	if !strings.Contains(output, "# TYPE mtws_connections_active gauge") {
		t.Fatalf("expected active connection gauge, got %q", output)
	}
	if !strings.Contains(output, "mtws_waf_blocks_total 1") {
		t.Fatalf("expected waf block metric, got %q", output)
	}
}
