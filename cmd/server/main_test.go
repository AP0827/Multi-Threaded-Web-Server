package main

import (
	"io"
	"net"
	"strings"
	"testing"

	"MTWS/core"
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

func issueRawRequest(t *testing.T, rawRequest string) string {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		core.HandleConnection(serverConn, buildRouter())
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
