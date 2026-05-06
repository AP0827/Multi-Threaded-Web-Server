package core

import (
	mtwshttp "MTWS/http"
	"encoding/json"
	"log"
	"net"
	"time"
)

type securityLogEntry struct {
	Time       string `json:"time"`
	Event      string `json:"event"`
	RemoteAddr string `json:"remote_addr"`
	Field      string `json:"field,omitempty"`
	Pattern    string `json:"pattern,omitempty"`
	RuleID     string `json:"rule_id,omitempty"`
	Status     int    `json:"status"`
	Reason     string `json:"reason,omitempty"`
}

type accessLogEntry struct {
	Time       string  `json:"time"`
	Event      string  `json:"event"`
	RemoteAddr string  `json:"remote_addr"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	Bytes      int     `json:"bytes"`
	DurationMS float64 `json:"duration_ms"`
}

func logSecurityEvent(conn net.Conn, field string, pattern string, ruleID string, status int) {
	writeStructuredLog(securityLogEntry{
		Time:       time.Now().UTC().Format(time.RFC3339Nano),
		Event:      "waf_block",
		RemoteAddr: remoteAddr(conn),
		Field:      field,
		Pattern:    pattern,
		RuleID:     ruleID,
		Status:     status,
	})
}

func logParseReject(conn net.Conn, reason string, status int) {
	writeStructuredLog(securityLogEntry{
		Time:       time.Now().UTC().Format(time.RFC3339Nano),
		Event:      "parse_reject",
		RemoteAddr: remoteAddr(conn),
		Status:     status,
		Reason:     reason,
	})
}

func logAccessEvent(conn net.Conn, req *mtwshttp.Request, status int, bytes int, duration time.Duration) {
	if req == nil {
		return
	}

	writeStructuredLog(accessLogEntry{
		Time:       time.Now().UTC().Format(time.RFC3339Nano),
		Event:      "access",
		RemoteAddr: remoteAddr(conn),
		Method:     req.Method(),
		Path:       req.Path(),
		Status:     status,
		Bytes:      bytes,
		DurationMS: float64(duration.Microseconds()) / 1000,
	})
}

func writeStructuredLog(entry any) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("structured_log_error=%v", err)
		return
	}
	log.Println(string(data))
}

func remoteAddr(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return ""
	}
	return conn.RemoteAddr().String()
}
