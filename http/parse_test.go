package mtwshttp

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseRequestGET(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n"

	req, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.Method() != "GET" {
		t.Fatalf("expected method GET, got %q", req.Method())
	}
	if req.Path() != "/" {
		t.Fatalf("expected path /, got %q", req.Path())
	}
	if req.Version() != "HTTP/1.1" {
		t.Fatalf("expected version HTTP/1.1, got %q", req.Version())
	}
	if got := req.Headers()["Host"]; got != "localhost" {
		t.Fatalf("expected Host localhost, got %q", got)
	}
	if len(req.Body()) != 0 {
		t.Fatalf("expected empty body, got %d bytes", len(req.Body()))
	}
}

func TestParseRequestPOSTWithBody(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\n\r\nhello"

	req, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.Method() != "POST" {
		t.Fatalf("expected method POST, got %q", req.Method())
	}
	if req.Path() != "/api" {
		t.Fatalf("expected path /api, got %q", req.Path())
	}
	if req.Body() != "hello" {
		t.Fatalf("expected body hello, got %q", req.Body())
	}
}

func TestParseRequestMalformedRequestLine(t *testing.T) {
	raw := "GET /\r\nHost: localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for malformed request line")
	}
}

func TestParseRequestUnsupportedVersion(t *testing.T) {
	raw := "GET / HTTP/1.0\r\nHost: localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for unsupported HTTP version")
	}
}

func TestParseRequestMalformedHeader(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for malformed header")
	}
}

func TestParseRequestMissingHost(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nUser-Agent: test\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for missing Host header")
	}
}

func TestParseRequestInvalidContentLength(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nContent-Length: abc\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid content-length")
	}
}

func TestParseRequestRejectsTransferEncoding(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for unsupported transfer-encoding")
	}
}

func TestParseRequestIncompleteBody(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\n\r\nabc"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for incomplete body")
	}
}

func TestParseRequestBlocksMaliciousURIPattern(t *testing.T) {
	raw := "GET /search?q=UNION%20SELECT HTTP/1.1\r\nHost: localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected waf error")
	}

	securityErr, ok := err.(*SecurityError)
	if !ok {
		t.Fatalf("expected SecurityError, got %T", err)
	}
	if securityErr.Field != "uri" {
		t.Fatalf("expected uri field, got %q", securityErr.Field)
	}
}

func TestParseRequestBlocksMaliciousHeaderPattern(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\nUser-Agent: <script>alert(1)</script>\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected waf error")
	}

	securityErr, ok := err.(*SecurityError)
	if !ok {
		t.Fatalf("expected SecurityError, got %T", err)
	}
	if securityErr.Field != "header:User-Agent" {
		t.Fatalf("expected header field, got %q", securityErr.Field)
	}
}

func TestParseRequestBlocksMaliciousBodyPattern(t *testing.T) {
	raw := "POST /submit HTTP/1.1\r\nHost: localhost\r\nContent-Length: 13\r\n\r\n<script>boom!"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected waf error")
	}

	securityErr, ok := err.(*SecurityError)
	if !ok {
		t.Fatalf("expected SecurityError, got %T", err)
	}
	if securityErr.Field != "body" {
		t.Fatalf("expected body field, got %q", securityErr.Field)
	}
}

func TestParseRequestRejectsDuplicateHeader(t *testing.T) {
	raw := "POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nContent-Length: 0\r\n\r\nhello"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for duplicate header")
	}
}

func TestParseRequestRejectsObsoleteLineFolding(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\nX-Test: first\r\n second-line\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for obsolete line folding")
	}
}

func TestParseRequestDoesNotMatchAcrossFields(t *testing.T) {
	raw := "GET /search?q=uni HTTP/1.1\r\nHost: localhost\r\nX-Test: on select\r\n\r\n"

	req, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("expected request to parse, got %v", err)
	}
	if req.Path() != "/search?q=uni" {
		t.Fatalf("unexpected path %q", req.Path())
	}
}
