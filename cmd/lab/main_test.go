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
