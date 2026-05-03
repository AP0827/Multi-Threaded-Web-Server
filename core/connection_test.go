package core

import (
	mtwshttp "MTWS/http"
	"io"
	"net"
	"strings"
	"testing"
	"time"
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

	rawRequest := "GET /health HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
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

	rawRequest := "GET /health\r\nHost: localhost\r\nConnection: close\r\n\r\n"
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

	rawRequest := "GET /?q=UNION%20SELECT HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
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

	rawRequest := "GET /search?q=harmless HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
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

func TestHandleConnectionKeepsHTTP11ConnectionAlive(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	router := NewRouter()
	router.Handle("/health", func(w *ResponseWriter, req *mtwshttp.Request) {
		if err := w.WriteText(StatusOK, "ok\n"); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		HandleConnectionWithOptions(serverConn, router, ConnectionOptions{
			MaxRequestsPerConnection: 2,
			IdleTimeout:              time.Second,
		})
	}()

	rawRequests := strings.Join([]string{
		"GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n",
		"GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n",
	}, "")
	writeDone := make(chan error, 1)
	go func() {
		_, err := clientConn.Write([]byte(rawRequests))
		writeDone <- err
	}()

	responseBytes, err := io.ReadAll(clientConn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if err := <-writeDone; err != nil {
		t.Fatalf("write requests: %v", err)
	}
	<-done

	response := string(responseBytes)
	if got := strings.Count(response, "HTTP/1.1 200 OK"); got != 2 {
		t.Fatalf("expected two 200 responses on one connection, got %d in %q", got, response)
	}
	if !strings.Contains(response, "Connection: keep-alive") {
		t.Fatalf("expected first response to keep the connection alive, got %q", response)
	}
	if !strings.Contains(response, "Connection: close") {
		t.Fatalf("expected final response to close after max requests, got %q", response)
	}
}
