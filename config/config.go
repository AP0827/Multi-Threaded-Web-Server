package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ServerAddress            = ":8080"
	WorkerPoolSize           = 10
	JobQueueSize             = 200
	DefaultRateLimitRate     = 5.0
	DefaultRateLimitCapacity = 10.0
	DefaultReadTimeout       = 5 * time.Second
	DefaultWriteTimeout      = 5 * time.Second
	DefaultIdleTimeout       = 30 * time.Second
	DefaultShutdownTimeout   = 10 * time.Second
	DefaultQueueTimeout      = 250 * time.Millisecond
	DefaultMaxKeepAlive      = 100
)

type Config struct {
	ServerAddress     string
	WorkerPoolSize    int
	JobQueueSize      int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	QueueTimeout      time.Duration
	MaxKeepAlive      int
	TLSCertFile       string
	TLSKeyFile        string
	RateLimitEnabled  bool
	RateLimitRate     float64
	RateLimitCapacity float64
}

func Load() Config {
	return Config{
		ServerAddress:     envString("MTWS_ADDR", ServerAddress),
		WorkerPoolSize:    envInt("MTWS_WORKERS", WorkerPoolSize),
		JobQueueSize:      envInt("MTWS_JOB_QUEUE_SIZE", JobQueueSize),
		ReadTimeout:       envDuration("MTWS_READ_TIMEOUT", DefaultReadTimeout),
		WriteTimeout:      envDuration("MTWS_WRITE_TIMEOUT", DefaultWriteTimeout),
		IdleTimeout:       envDuration("MTWS_IDLE_TIMEOUT", DefaultIdleTimeout),
		ShutdownTimeout:   envDuration("MTWS_SHUTDOWN_TIMEOUT", DefaultShutdownTimeout),
		QueueTimeout:      envDuration("MTWS_QUEUE_TIMEOUT", DefaultQueueTimeout),
		MaxKeepAlive:      envInt("MTWS_MAX_KEEPALIVE_REQUESTS", DefaultMaxKeepAlive),
		TLSCertFile:       strings.TrimSpace(os.Getenv("MTWS_TLS_CERT_FILE")),
		TLSKeyFile:        strings.TrimSpace(os.Getenv("MTWS_TLS_KEY_FILE")),
		RateLimitEnabled:  RateLimitEnabled(),
		RateLimitRate:     RateLimitRate(),
		RateLimitCapacity: RateLimitCapacity(),
	}
}

func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

func (c Config) TLSPartiallyConfigured() bool {
	return (c.TLSCertFile == "") != (c.TLSKeyFile == "")
}

func RateLimitEnabled() bool {
	return !envBool("MTWS_RATE_LIMIT_DISABLED") && !envBool("MTWS_BENCHMARK_MODE")
}

func RateLimitRate() float64 {
	return envPositiveFloat("MTWS_RATE_LIMIT_RATE", DefaultRateLimitRate)
}

func RateLimitCapacity() float64 {
	return envPositiveFloat("MTWS_RATE_LIMIT_CAPACITY", DefaultRateLimitCapacity)
}

func envString(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(name string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envFloat(name string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		log.Printf("Invalid %s=%q; using %.2f", name, value, fallback)
		return fallback
	}

	return parsed
}

func envPositiveFloat(name string, fallback float64) float64 {
	parsed := envFloat(name, fallback)
	if parsed <= 0 {
		log.Printf("Invalid %s; using %.2f because the value must be positive", name, fallback)
		return fallback
	}
	return parsed
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid %s=%q; using %d", name, value, fallback)
		return fallback
	}

	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid %s=%q; using %s", name, value, fallback)
		return fallback
	}

	return parsed
}
