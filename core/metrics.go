package core

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

type Metrics struct {
	acceptedConnections  atomic.Uint64
	activeConnections    atomic.Uint64
	totalRequests        atomic.Uint64
	totalRequestDuration atomic.Uint64
	maxRequestDuration   atomic.Uint64
	wafBlocks            atomic.Uint64
	parseRejects         atomic.Uint64
	queueRejects         atomic.Uint64
	rateLimited          atomic.Uint64
	responses2xx         atomic.Uint64
	responses3xx         atomic.Uint64
	responses4xx         atomic.Uint64
	responses5xx         atomic.Uint64
}

type MetricsSnapshot struct {
	AcceptedConnections      uint64  `json:"accepted_connections"`
	ActiveConnections        uint64  `json:"active_connections"`
	TotalRequests            uint64  `json:"total_requests"`
	AverageRequestDurationMS float64 `json:"average_request_duration_ms"`
	MaxRequestDurationMS     float64 `json:"max_request_duration_ms"`
	WAFBlocks                uint64  `json:"waf_blocks"`
	ParseRejects             uint64  `json:"parse_rejects"`
	QueueRejects             uint64  `json:"queue_rejects"`
	RateLimited              uint64  `json:"rate_limited"`
	Responses2xx             uint64  `json:"responses_2xx"`
	Responses3xx             uint64  `json:"responses_3xx"`
	Responses4xx             uint64  `json:"responses_4xx"`
	Responses5xx             uint64  `json:"responses_5xx"`
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) IncAcceptedConnection() {
	if m != nil {
		m.acceptedConnections.Add(1)
	}
}

func (m *Metrics) IncActiveConnection() {
	if m != nil {
		m.activeConnections.Add(1)
	}
}

func (m *Metrics) DecActiveConnection() {
	if m != nil {
		m.activeConnections.Add(^uint64(0))
	}
}

func (m *Metrics) IncRequest() {
	if m != nil {
		m.totalRequests.Add(1)
	}
}

func (m *Metrics) RecordRequestDuration(duration time.Duration) {
	if m == nil || duration < 0 {
		return
	}

	micros := uint64(duration / time.Microsecond)
	m.totalRequestDuration.Add(micros)

	for {
		current := m.maxRequestDuration.Load()
		if micros <= current {
			return
		}
		if m.maxRequestDuration.CompareAndSwap(current, micros) {
			return
		}
	}
}

func (m *Metrics) IncWAFBlock() {
	if m != nil {
		m.wafBlocks.Add(1)
	}
}

func (m *Metrics) IncParseReject() {
	if m != nil {
		m.parseRejects.Add(1)
	}
}

func (m *Metrics) IncQueueReject() {
	if m != nil {
		m.queueRejects.Add(1)
	}
}

func (m *Metrics) IncRateLimited() {
	if m != nil {
		m.rateLimited.Add(1)
	}
}

func (m *Metrics) RecordResponse(status int) {
	if m == nil {
		return
	}

	switch {
	case status >= 200 && status < 300:
		m.responses2xx.Add(1)
	case status >= 300 && status < 400:
		m.responses3xx.Add(1)
	case status >= 400 && status < 500:
		m.responses4xx.Add(1)
	case status >= 500:
		m.responses5xx.Add(1)
	}
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}

	return MetricsSnapshot{
		AcceptedConnections:      m.acceptedConnections.Load(),
		ActiveConnections:        m.activeConnections.Load(),
		TotalRequests:            m.totalRequests.Load(),
		AverageRequestDurationMS: averageDurationMS(m.totalRequestDuration.Load(), m.totalRequests.Load()),
		MaxRequestDurationMS:     float64(m.maxRequestDuration.Load()) / 1000,
		WAFBlocks:                m.wafBlocks.Load(),
		ParseRejects:             m.parseRejects.Load(),
		QueueRejects:             m.queueRejects.Load(),
		RateLimited:              m.rateLimited.Load(),
		Responses2xx:             m.responses2xx.Load(),
		Responses3xx:             m.responses3xx.Load(),
		Responses4xx:             m.responses4xx.Load(),
		Responses5xx:             m.responses5xx.Load(),
	}
}

func (m *Metrics) Prometheus() string {
	snapshot := m.Snapshot()
	var b strings.Builder

	writeMetric(&b, "mtws_connections_accepted_total", "Total TCP connections accepted.", snapshot.AcceptedConnections)
	writeGaugeUint64(&b, "mtws_connections_active", "Currently active TCP connections.", snapshot.ActiveConnections)
	writeMetric(&b, "mtws_requests_total", "Total parsed HTTP requests.", snapshot.TotalRequests)
	writeGaugeFloat(&b, "mtws_request_duration_average_ms", "Average request duration in milliseconds.", snapshot.AverageRequestDurationMS)
	writeGaugeFloat(&b, "mtws_request_duration_max_ms", "Maximum request duration in milliseconds.", snapshot.MaxRequestDurationMS)
	writeMetric(&b, "mtws_waf_blocks_total", "Total requests blocked by in-parser WAF rules.", snapshot.WAFBlocks)
	writeMetric(&b, "mtws_parse_rejects_total", "Total malformed requests rejected by the strict parser.", snapshot.ParseRejects)
	writeMetric(&b, "mtws_queue_rejects_total", "Total connections rejected because the worker queue was saturated.", snapshot.QueueRejects)
	writeMetric(&b, "mtws_rate_limited_total", "Total connections rejected by the token-bucket rate limiter.", snapshot.RateLimited)
	writeMetric(&b, "mtws_responses_2xx_total", "Total 2xx responses.", snapshot.Responses2xx)
	writeMetric(&b, "mtws_responses_3xx_total", "Total 3xx responses.", snapshot.Responses3xx)
	writeMetric(&b, "mtws_responses_4xx_total", "Total 4xx responses.", snapshot.Responses4xx)
	writeMetric(&b, "mtws_responses_5xx_total", "Total 5xx responses.", snapshot.Responses5xx)

	return b.String()
}

func writeMetric(b *strings.Builder, name string, help string, value uint64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s counter\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}

func writeGaugeUint64(b *strings.Builder, name string, help string, value uint64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s gauge\n", name)
	fmt.Fprintf(b, "%s %d\n", name, value)
}

func writeGaugeFloat(b *strings.Builder, name string, help string, value float64) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s gauge\n", name)
	fmt.Fprintf(b, "%s %.3f\n", name, value)
}

func averageDurationMS(totalMicros uint64, totalRequests uint64) float64 {
	if totalMicros == 0 || totalRequests == 0 {
		return 0
	}
	return float64(totalMicros) / 1000 / float64(totalRequests)
}
