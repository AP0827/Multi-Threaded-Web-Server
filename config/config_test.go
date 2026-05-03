package config

import (
	"testing"
	"time"
)

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

func TestRateLimitRejectsZeroOverride(t *testing.T) {
	t.Setenv("MTWS_RATE_LIMIT_RATE", "0")
	t.Setenv("MTWS_RATE_LIMIT_CAPACITY", "0")

	if RateLimitRate() != DefaultRateLimitRate {
		t.Fatalf("expected default rate, got %.2f", RateLimitRate())
	}
	if RateLimitCapacity() != DefaultRateLimitCapacity {
		t.Fatalf("expected default capacity, got %.2f", RateLimitCapacity())
	}
}

func TestLoadReadsOperationalOverrides(t *testing.T) {
	t.Setenv("MTWS_ADDR", "127.0.0.1:9090")
	t.Setenv("MTWS_WORKERS", "4")
	t.Setenv("MTWS_JOB_QUEUE_SIZE", "12")
	t.Setenv("MTWS_READ_TIMEOUT", "2s")
	t.Setenv("MTWS_WRITE_TIMEOUT", "3s")
	t.Setenv("MTWS_IDLE_TIMEOUT", "30s")
	t.Setenv("MTWS_SHUTDOWN_TIMEOUT", "4s")
	t.Setenv("MTWS_QUEUE_TIMEOUT", "150ms")
	t.Setenv("MTWS_MAX_KEEPALIVE_REQUESTS", "77")
	t.Setenv("MTWS_TLS_CERT_FILE", "server.crt")
	t.Setenv("MTWS_TLS_KEY_FILE", "server.key")

	cfg := Load()
	if cfg.ServerAddress != "127.0.0.1:9090" {
		t.Fatalf("expected address override, got %q", cfg.ServerAddress)
	}
	if cfg.WorkerPoolSize != 4 {
		t.Fatalf("expected worker override, got %d", cfg.WorkerPoolSize)
	}
	if cfg.JobQueueSize != 12 {
		t.Fatalf("expected queue override, got %d", cfg.JobQueueSize)
	}
	if cfg.ReadTimeout != 2*time.Second {
		t.Fatalf("expected read timeout override, got %s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 3*time.Second {
		t.Fatalf("expected write timeout override, got %s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 30*time.Second {
		t.Fatalf("expected idle timeout override, got %s", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 4*time.Second {
		t.Fatalf("expected shutdown timeout override, got %s", cfg.ShutdownTimeout)
	}
	if cfg.QueueTimeout != 150*time.Millisecond {
		t.Fatalf("expected queue timeout override, got %s", cfg.QueueTimeout)
	}
	if cfg.MaxKeepAlive != 77 {
		t.Fatalf("expected max keep-alive override, got %d", cfg.MaxKeepAlive)
	}
	if !cfg.TLSEnabled() {
		t.Fatal("expected TLS to be enabled when both files are configured")
	}
}

func TestTLSPartialConfiguration(t *testing.T) {
	cfg := Config{TLSCertFile: "server.crt"}
	if !cfg.TLSPartiallyConfigured() {
		t.Fatal("expected partial TLS configuration")
	}

	cfg.TLSKeyFile = "server.key"
	if cfg.TLSPartiallyConfigured() {
		t.Fatal("expected complete TLS configuration")
	}
}
