package mtwshttp

import (
	"MTWS/security/waf"
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

const (
	maxRequestLineBytes = 4096
	maxHeaderLineBytes  = 8192
	maxHeaderCount      = 64
	maxBodyBytes        = 1 << 20 // 1 MiB
)

var defaultWAF = waf.NewDefaultAutomaton()

// ParseRequest is an exported wrapper so other packages (like core) can use the parser.
func ParseRequest(reader *bufio.Reader) (*Request, error) {
	return parseRequest(reader, defaultWAF)
}

func parseRequest(reader *bufio.Reader, automaton *waf.Automaton) (*Request, error) {
	if reader == nil {
		return nil, fmt.Errorf("nil reader")
	}

	method, path, version, err := readRequestLine(reader, automaton)
	if err != nil {
		return nil, err
	}

	req := &Request{
		method:  method,
		path:    path,
		version: version,
		headers: make(map[string]string),
	}

	if req.version != "HTTP/1.1" {
		return nil, fmt.Errorf("unsupported HTTP version: %s", req.version)
	}

	seenHeaders := make(map[string]struct{})
	for i := 0; ; i++ {
		if i >= maxHeaderCount {
			return nil, fmt.Errorf("too many headers")
		}

		name, value, done, err := readHeaderLine(reader, automaton)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
		if _, exists := seenHeaders[name]; exists {
			return nil, fmt.Errorf("duplicate header: %s", name)
		}
		seenHeaders[name] = struct{}{}
		req.headers[name] = value
	}

	contentLength := 0
	if transferEncoding, ok := req.headers["Transfer-Encoding"]; ok {
		return nil, fmt.Errorf("unsupported transfer-encoding: %q", transferEncoding)
	}

	host, ok := req.headers["Host"]
	if !ok || strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("missing required Host header")
	}

	if cl, ok := req.headers["Content-Length"]; ok {
		parsedLen, err := strconv.Atoi(cl)
		if err != nil || parsedLen < 0 {
			return nil, fmt.Errorf("invalid content-length: %q", cl)
		}
		if parsedLen > maxBodyBytes {
			return nil, fmt.Errorf("request body too large")
		}
		contentLength = parsedLen
	}

	if contentLength > 0 {
		req.body = make([]byte, contentLength)
		if err := readBody(reader, req.body, automaton); err != nil {
			return nil, err
		}
	}

	return req, nil
}

func readRequestLine(reader *bufio.Reader, automaton *waf.Automaton) (string, string, string, error) {
	line, err := readLineLimited(reader, maxRequestLineBytes)
	if err != nil {
		return "", "", "", err
	}

	parts := strings.Fields(string(line))
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("malformed request line")
	}

	if match := scanField(automaton, "uri", []byte(parts[1])); match != nil {
		return "", "", "", &SecurityError{Pattern: match.Pattern, Field: match.Field}
	}

	return parts[0], parts[1], parts[2], nil
}

func readHeaderLine(reader *bufio.Reader, automaton *waf.Automaton) (string, string, bool, error) {
	line, err := readLineLimited(reader, maxHeaderLineBytes)
	if err != nil {
		return "", "", false, err
	}
	if len(line) == 0 {
		return "", "", true, nil
	}
	if line[0] == ' ' || line[0] == '\t' {
		return "", "", false, fmt.Errorf("obsolete line folding is not supported")
	}

	name, value, ok := strings.Cut(string(line), ":")
	if !ok {
		return "", "", false, fmt.Errorf("malformed header: %q", string(line))
	}

	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)
	if name == "" {
		return "", "", false, fmt.Errorf("empty header name")
	}

	canonicalName := textproto.CanonicalMIMEHeaderKey(name)
	fieldName := "header:" + canonicalName
	if match := scanField(automaton, fieldName, []byte(value)); match != nil {
		return "", "", false, &SecurityError{Pattern: match.Pattern, Field: match.Field}
	}

	return canonicalName, value, false, nil
}

func scanField(automaton *waf.Automaton, field string, value []byte) *waf.Match {
	if automaton == nil || len(value) == 0 {
		return nil
	}

	if field == "uri" {
		decoded, err := url.PathUnescape(string(value))
		if err == nil {
			value = []byte(decoded)
		}
	}

	return automaton.NewScanner().Feed(field, value)
}

func readBody(reader *bufio.Reader, body []byte, automaton *waf.Automaton) error {
	if len(body) == 0 {
		return nil
	}

	scanner := automaton.NewScanner()
	offset := 0
	for offset < len(body) {
		n, err := reader.Read(body[offset:])
		if n > 0 {
			if match := scanner.Feed("body", body[offset:offset+n]); match != nil {
				return &SecurityError{Pattern: match.Pattern, Field: match.Field}
			}
			offset += n
		}

		if err != nil {
			if err == io.EOF && offset == len(body) {
				break
			}
			if err == io.EOF {
				return fmt.Errorf("incomplete request body: %w", err)
			}
			return err
		}
	}

	return nil
}

func readLineLimited(reader *bufio.Reader, maxBytes int) ([]byte, error) {
	line := make([]byte, 0, maxBytes)

	for {
		b, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return nil, fmt.Errorf("incomplete line")
			}
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected end of request")
			}
			return nil, err
		}

		if len(line) >= maxBytes {
			return nil, fmt.Errorf("line exceeds %d bytes", maxBytes)
		}

		line = append(line, b)
		if len(line) >= 2 && line[len(line)-2] == '\r' && line[len(line)-1] == '\n' {
			return line[:len(line)-2], nil
		}
	}
}
