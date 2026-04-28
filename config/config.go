package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	ServerAddress            = ":8080"
	WorkerPoolSize           = 10
	JobQueueSize             = 200
	DefaultRateLimitRate     = 5.0
	DefaultRateLimitCapacity = 10.0
)

func RateLimitEnabled() bool {
	return !envBool("MTWS_RATE_LIMIT_DISABLED") && !envBool("MTWS_BENCHMARK_MODE")
}

func RateLimitRate() float64 {
	return envFloat("MTWS_RATE_LIMIT_RATE", DefaultRateLimitRate)
}

func RateLimitCapacity() float64 {
	return envFloat("MTWS_RATE_LIMIT_CAPACITY", DefaultRateLimitCapacity)
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
