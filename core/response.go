package core

import (
	"errors"
	"fmt"
	"net"
	"time"
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
	StatusMethodNotAllowed    = StatusCode{Code: 405, Reason: "Method Not Allowed", Body: "Method Not Allowed\n"}
	StatusTooManyRequests     = StatusCode{Code: 429, Reason: "Too Many Requests", Body: "Too Many Requests\n"}
	StatusInternalServerError = StatusCode{Code: 500, Reason: "Internal Server Error", Body: "Internal Server Error\n"}
	StatusServiceUnavailable  = StatusCode{Code: 503, Reason: "Service Unavailable", Body: "Service Unavailable\n"}
)

type ResponseOptions struct {
	WriteTimeout         time.Duration
	Metrics              *Metrics
	KeepAlive            bool
	KeepAliveTimeout     time.Duration
	KeepAliveMaxRequests int
}

type ResponseWriter struct {
	conn                 net.Conn
	httpVersion          string
	writeTimeout         time.Duration
	metrics              *Metrics
	keepAlive            bool
	keepAliveTimeout     time.Duration
	keepAliveMaxRequests int
	wrote                bool
	statusCode           int
	bytes                int
}

func NewResponseWriter(conn net.Conn, httpVersion string) *ResponseWriter {
	return NewResponseWriterWithOptions(conn, httpVersion, ResponseOptions{})
}

func NewResponseWriterWithOptions(conn net.Conn, httpVersion string, options ResponseOptions) *ResponseWriter {
	if httpVersion == "" {
		httpVersion = "HTTP/1.1"
	}

	return &ResponseWriter{
		conn:                 conn,
		httpVersion:          httpVersion,
		writeTimeout:         options.WriteTimeout,
		metrics:              options.Metrics,
		keepAlive:            options.KeepAlive,
		keepAliveTimeout:     options.KeepAliveTimeout,
		keepAliveMaxRequests: options.KeepAliveMaxRequests,
	}
}

func (w *ResponseWriter) WriteText(status StatusCode, body string) error {
	if body == "" {
		body = status.Body
	}
	return w.write(status, "text/plain; charset=utf-8", []byte(body))
}

func (w *ResponseWriter) WriteBytes(status StatusCode, contentType string, body []byte) error {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return w.write(status, contentType, body)
}

func (w *ResponseWriter) write(status StatusCode, contentType string, body []byte) error {
	if w == nil || w.conn == nil {
		return errors.New("response writer is not configured")
	}
	if w.wrote {
		return errors.New("response already written")
	}
	if w.writeTimeout > 0 {
		if err := w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout)); err != nil {
			return err
		}
	}

	connectionHeader := "close"
	keepAliveHeader := ""
	if w.keepAlive {
		connectionHeader = "keep-alive"
		keepAliveHeader = fmt.Sprintf("Keep-Alive: timeout=%d, max=%d\r\n", int(w.keepAliveTimeout.Seconds()), w.keepAliveMaxRequests)
	}

	header := fmt.Sprintf(
		"%s %d %s\r\nDate: %s\r\nServer: MTWS\r\nContent-Type: %s\r\nX-Content-Type-Options: nosniff\r\nContent-Length: %d\r\nConnection: %s\r\n%s\r\n",
		w.httpVersion,
		status.Code,
		status.Reason,
		time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT"),
		contentType,
		len(body),
		connectionHeader,
		keepAliveHeader,
	)

	response := append([]byte(header), body...)
	n, err := w.conn.Write(response)
	if err != nil {
		return err
	}
	w.wrote = true
	w.statusCode = status.Code
	w.bytes = n
	w.metrics.RecordResponse(status.Code)
	return nil
}

func writeErrorResponse(conn net.Conn, httpVersion string, status StatusCode) {
	writeErrorResponseWithOptions(conn, httpVersion, status, ResponseOptions{})
}

func writeErrorResponseWithOptions(conn net.Conn, httpVersion string, status StatusCode, options ResponseOptions) {
	writer := NewResponseWriterWithOptions(conn, httpVersion, options)
	_ = writer.WriteText(status, status.Body)
}

func (w *ResponseWriter) Wrote() bool {
	return w != nil && w.wrote
}

func (w *ResponseWriter) StatusCode() int {
	if w == nil || w.statusCode == 0 {
		return 0
	}
	return w.statusCode
}

func (w *ResponseWriter) BytesWritten() int {
	if w == nil {
		return 0
	}
	return w.bytes
}
