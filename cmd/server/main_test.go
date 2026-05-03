package main

import (
	"MTWS/config"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"MTWS/core"
	"MTWS/pool"
)

func TestBuildRouterServesSubmit(t *testing.T) {
	response := issueRawRequest(t, "POST /submit HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n6\r\n world\r\n0\r\n\r\n")
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Fatalf("expected 200 OK response, got %q", response)
	}
	if !strings.HasSuffix(response, "MTWS baseline server is running\n") {
		t.Fatalf("expected submit body, got %q", response)
	}
}

func TestBuildRouterServesSearch(t *testing.T) {
	response := issueRawRequest(t, "GET /search?q=harmless HTTP/1.1\r\nHost: localhost\r\n\r\n")
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Fatalf("expected 200 OK response, got %q", response)
	}
	if !strings.HasSuffix(response, "query=q=harmless\n") {
		t.Fatalf("expected query echo body, got %q", response)
	}
}

func TestBuildRouterServesReadiness(t *testing.T) {
	response := issueRawRequest(t, "GET /ready HTTP/1.1\r\nHost: localhost\r\n\r\n")
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Fatalf("expected 200 OK response, got %q", response)
	}
	if !strings.HasSuffix(response, "ready\n") {
		t.Fatalf("expected ready body, got %q", response)
	}
}

func TestBuildRouterServesMetrics(t *testing.T) {
	response := issueRawRequest(t, "GET /metrics HTTP/1.1\r\nHost: localhost\r\n\r\n")
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Fatalf("expected 200 OK response, got %q", response)
	}
	if !strings.Contains(response, "mtws_requests_total") {
		t.Fatalf("expected prometheus metrics body, got %q", response)
	}
}

func TestBuildRouterRejectsWrongMethod(t *testing.T) {
	response := issueRawRequest(t, "GET /submit HTTP/1.1\r\nHost: localhost\r\n\r\n")
	if !strings.Contains(response, "HTTP/1.1 405 Method Not Allowed") {
		t.Fatalf("expected 405 response, got %q", response)
	}
}

func TestEnqueueJobTimesOutWhenQueueIsFull(t *testing.T) {
	jobs := make(chan pool.Job, 1)
	jobs <- pool.Job{}

	ok := enqueueJob(context.Background(), jobs, pool.Job{}, time.Millisecond)
	if ok {
		t.Fatal("expected enqueue to fail when the queue stays full past the timeout")
	}
}

func TestRunRejectsPartialTLSConfiguration(t *testing.T) {
	cfg := config.Load()
	cfg.TLSCertFile = "cert.pem"
	cfg.TLSKeyFile = ""

	err := run(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "TLS requires both") {
		t.Fatalf("expected partial TLS config error, got %v", err)
	}
}

func issueRawRequest(t *testing.T, rawRequest string) string {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		core.HandleConnectionWithOptions(serverConn, buildRouter(), core.ConnectionOptions{
			DisableKeepAlive: true,
		})
	}()

	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	<-done
	return string(responseBytes)
}
