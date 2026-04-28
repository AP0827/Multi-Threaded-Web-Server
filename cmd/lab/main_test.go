package main

import (
	"testing"
	"time"
)

func TestSummarizeLatencies(t *testing.T) {
	summary := summarizeLatencies([]time.Duration{
		10 * time.Millisecond,
		40 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		50 * time.Millisecond,
	})

	if summary.Min != 10*time.Millisecond {
		t.Fatalf("expected min 10ms, got %s", summary.Min)
	}
	if summary.P50 != 30*time.Millisecond {
		t.Fatalf("expected p50 30ms, got %s", summary.P50)
	}
	if summary.P95 != 50*time.Millisecond {
		t.Fatalf("expected p95 50ms, got %s", summary.P95)
	}
	if summary.Max != 50*time.Millisecond {
		t.Fatalf("expected max 50ms, got %s", summary.Max)
	}
}

func TestNormalizeHTTPFixture(t *testing.T) {
	normalized := string(normalizeHTTPFixture([]byte("GET / HTTP/1.1\nHost: localhost\n\n")))
	expected := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if normalized != expected {
		t.Fatalf("expected %q, got %q", expected, normalized)
	}
}

func TestSummarizeBenchmarkReports(t *testing.T) {
	summary, err := summarizeBenchmarkReports([]benchmarkReport{
		{
			URL:        "http://127.0.0.1:8080/health",
			Throughput: 100,
			Latency: latencyStats{
				Average: "10ms",
				P50:     "9ms",
				P95:     "20ms",
				P99:     "30ms",
			},
		},
		{
			URL:        "http://127.0.0.1:8080/health",
			Throughput: 200,
			Latency: latencyStats{
				Average: "20ms",
				P50:     "19ms",
				P95:     "30ms",
				P99:     "40ms",
			},
		},
		{
			URL:        "http://127.0.0.1:8080/health",
			Throughput: 300,
			Latency: latencyStats{
				Average: "30ms",
				P50:     "29ms",
				P95:     "40ms",
				P99:     "50ms",
			},
		},
	})
	if err != nil {
		t.Fatalf("summarize reports: %v", err)
	}
	if summary.Runs != 3 {
		t.Fatalf("expected 3 runs, got %d", summary.Runs)
	}
	if summary.Throughput != 200 {
		t.Fatalf("expected median throughput 200, got %.2f", summary.Throughput)
	}
	if summary.P95 != 30*time.Millisecond {
		t.Fatalf("expected median p95 30ms, got %s", summary.P95)
	}
}
