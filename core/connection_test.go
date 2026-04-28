package core

import (
	mtwshttp "MTWS/http"
	"io"
	"net"
	"strings"
	"testing"
)

func TestHandleConnectionRoutesParsedRequest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	router := NewRouter()
	router.Handle("/health", func(w *ResponseWriter, req *mtwshttp.Request) {
		if req.Method() != "GET" {
			t.Fatalf("expected GET, got %s", req.Method())
		}
		if err := w.WriteText(StatusOK, "ok\n"); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		HandleConnection(serverConn, router)
	}()

	rawRequest := "GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	<-done

	response := string(responseBytes)
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Fatalf("expected 200 OK response, got %q", response)
	}
	if !strings.HasSuffix(response, "ok\n") {
		t.Fatalf("expected body ok, got %q", response)
	}
}

func TestHandleConnectionRejectsMalformedRequest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		HandleConnection(serverConn, NewRouter())
	}()

	rawRequest := "GET /health\r\nHost: localhost\r\n\r\n"
	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	<-done

	response := string(responseBytes)
	if !strings.Contains(response, "HTTP/1.1 400 Bad Request") {
		t.Fatalf("expected 400 response, got %q", response)
	}
}

func TestHandleConnectionBlocksMaliciousRequest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		HandleConnection(serverConn, NewRouter())
	}()

	rawRequest := "GET /?q=UNION%20SELECT HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	<-done

	response := string(responseBytes)
	if !strings.Contains(response, "HTTP/1.1 403 Forbidden") {
		t.Fatalf("expected 403 response, got %q", response)
	}
}

func TestHandleConnectionRoutesRequestWithQueryString(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	router := NewRouter()
	router.Handle("/search", func(w *ResponseWriter, req *mtwshttp.Request) {
		if err := w.WriteText(StatusOK, "matched\n"); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		HandleConnection(serverConn, router)
	}()

	rawRequest := "GET /search?q=harmless HTTP/1.1\r\nHost: localhost\r\n\r\n"
	if _, err := clientConn.Write([]byte(rawRequest)); err != nil {
		t.Fatalf("write request: %v", err)
	}

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	<-done

	response := string(responseBytes)
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Fatalf("expected 200 OK response, got %q", response)
	}
	if !strings.HasSuffix(response, "matched\n") {
		t.Fatalf("expected body matched, got %q", response)
	}
}
