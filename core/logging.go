package core

import (
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

func writeStructuredLog(entry securityLogEntry) {
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
