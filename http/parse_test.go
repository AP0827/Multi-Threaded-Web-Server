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

func TestParseRequestChunkedBody(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n6\r\n world\r\n0\r\nX-Trace: done\r\n\r\n"

	req, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if req.Body() != "hello world" {
		t.Fatalf("expected chunked body, got %q", req.Body())
	}
	if req.Trailers()["X-Trace"] != "done" {
		t.Fatalf("expected trailer X-Trace=done, got %q", req.Trailers()["X-Trace"])
	}
}

func TestParseRequestMalformedRequestLine(t *testing.T) {
	raw := "GET /\r\nHost: localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for malformed request line")
	}
}

func TestParseRequestInvalidMethodToken(t *testing.T) {
	raw := "GE T / HTTP/1.1\r\nHost: localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid method token")
	}
}

func TestParseRequestRejectsAbsoluteFormTarget(t *testing.T) {
	raw := "GET http://example.com/ HTTP/1.1\r\nHost: localhost\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for unsupported absolute-form target")
	}
}

func TestParseRequestAllowsOptionsAsterisk(t *testing.T) {
	raw := "OPTIONS * HTTP/1.1\r\nHost: localhost\r\n\r\n"

	req, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("expected OPTIONS * to parse, got %v", err)
	}
	if req.Method() != "OPTIONS" || req.Path() != "*" {
		t.Fatalf("unexpected request method=%q path=%q", req.Method(), req.Path())
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

func TestParseRequestInvalidHeaderName(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\nBad Header: value\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid header name")
	}
}

func TestParseRequestMissingHost(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nUser-Agent: test\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for missing Host header")
	}
}

func TestParseRequestInvalidHostWhitespace(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: local host\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid Host header")
	}
}

func TestParseRequestInvalidContentLength(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nContent-Length: abc\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid content-length")
	}
}

func TestParseRequestRejectsSignedContentLength(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nContent-Length: +5\r\n\r\nhello"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for signed content-length")
	}
}

func TestParseRequestRejectsTransferEncoding(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: gzip\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for unsupported transfer-encoding")
	}
}

func TestParseRequestRejectsContentLengthWithTransferEncoding(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for content-length with transfer-encoding")
	}
}

func TestParseRequestRejectsBadChunkTerminator(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\n0\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for invalid chunk terminator")
	}
}

func TestParseRequestRejectsForbiddenTrailer(t *testing.T) {
	raw := "POST /api HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n0\r\nContent-Length: 5\r\n\r\n"

	_, err := ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	if err == nil {
		t.Fatal("expected error for forbidden trailer")
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

func TestParseRequestBlocksSQLCommentObfuscation(t *testing.T) {
	raw := "GET /search?q=UNION/**/SELECT HTTP/1.1\r\nHost: localhost\r\n\r\n"

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
	if securityErr.RuleID == "" {
		t.Fatal("expected rule id")
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

func TestParseRequestBlocksMaliciousChunkedBodyPattern(t *testing.T) {
	raw := "POST /submit HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n4\r\n<scr\r\n9\r\nipt>boom!\r\n0\r\n\r\n"

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
