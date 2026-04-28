package config

import "testing"

func TestRateLimitEnabledDefaultsToTrue(t *testing.T) {
	t.Setenv("MTWS_RATE_LIMIT_DISABLED", "")
	t.Setenv("MTWS_BENCHMARK_MODE", "")

	if !RateLimitEnabled() {
		t.Fatal("expected rate limit to be enabled by default")
	}
}

func TestRateLimitCanBeDisabled(t *testing.T) {
	t.Setenv("MTWS_RATE_LIMIT_DISABLED", "true")
	t.Setenv("MTWS_BENCHMARK_MODE", "")

	if RateLimitEnabled() {
		t.Fatal("expected rate limit to be disabled")
	}
}

func TestBenchmarkModeDisablesRateLimit(t *testing.T) {
	t.Setenv("MTWS_RATE_LIMIT_DISABLED", "")
	t.Setenv("MTWS_BENCHMARK_MODE", "1")

	if RateLimitEnabled() {
		t.Fatal("expected benchmark mode to disable rate limit")
	}
}

func TestRateLimitOverrides(t *testing.T) {
	t.Setenv("MTWS_RATE_LIMIT_RATE", "123.5")
	t.Setenv("MTWS_RATE_LIMIT_CAPACITY", "456.5")

	if RateLimitRate() != 123.5 {
		t.Fatalf("expected rate override, got %.2f", RateLimitRate())
	}
	if RateLimitCapacity() != 456.5 {
		t.Fatalf("expected capacity override, got %.2f", RateLimitCapacity())
	}
}
