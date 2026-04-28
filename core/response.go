package core

import (
	"fmt"
	"net"
)

type StatusCode struct {
	Code   int
	Reason string
	Body   string
}

var (
	StatusOK                  = StatusCode{Code: 200, Reason: "OK", Body: "OK\n"}
	StatusBadRequest          = StatusCode{Code: 400, Reason: "Bad Request", Body: "Bad Request\n"}
	StatusForbidden           = StatusCode{Code: 403, Reason: "Forbidden", Body: "Forbidden\n"}
	StatusNotFound            = StatusCode{Code: 404, Reason: "Not Found", Body: "Not Found\n"}
	StatusTooManyRequests     = StatusCode{Code: 429, Reason: "Too Many Requests", Body: "Too Many Requests\n"}
	StatusInternalServerError = StatusCode{Code: 500, Reason: "Internal Server Error", Body: "Internal Server Error\n"}
)

type ResponseWriter struct {
	conn        net.Conn
	httpVersion string
}

func NewResponseWriter(conn net.Conn, httpVersion string) *ResponseWriter {
	if httpVersion == "" {
		httpVersion = "HTTP/1.1"
	}

	return &ResponseWriter{
		conn:        conn,
		httpVersion: httpVersion,
	}
}

func (w *ResponseWriter) WriteText(status StatusCode, body string) error {
	if body == "" {
		body = status.Body
	}

	response := fmt.Sprintf(
		"%s %d %s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		w.httpVersion,
		status.Code,
		status.Reason,
		len(body),
		body,
	)

	_, err := w.conn.Write([]byte(response))
	return err
}

func writeErrorResponse(conn net.Conn, httpVersion string, status StatusCode) {
	writer := NewResponseWriter(conn, httpVersion)
	_ = writer.WriteText(status, status.Body)
}
