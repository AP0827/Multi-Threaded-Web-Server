package mtwshttp

import (
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"strings"
)

const (
	maxRequestLineBytes = 4096
	maxHeaderLineBytes  = 8192
	maxHeaderCount      = 64
	maxBodyBytes        = 1 << 20 // 1 MiB
)

// ParseRequest is an exported wrapper so other packages (like core) can use the parser.
func ParseRequest(reader *bufio.Reader) (*Request, error) {
	return parseRequest(reader)
}

func parseRequest(reader *bufio.Reader) (*Request, error) {
	if reader == nil {
		return nil, fmt.Errorf("nil reader")
	}

	requestLine, err := readLine(reader, maxRequestLineBytes)
	if err != nil {
		return nil, err
	}

	parts := strings.Fields(requestLine)
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed request line")
	}

	req := &Request{
		method:  parts[0],
		path:    parts[1],
		version: parts[2],
		headers: make(map[string]string),
	}

	if req.version != "HTTP/1.1" {
		return nil, fmt.Errorf("unsupported HTTP version: %s", req.version)
	}

	for i := 0; ; i++ {
		if i >= maxHeaderCount {
			return nil, fmt.Errorf("too many headers")
		}

		headerLine, err := readLine(reader, maxHeaderLineBytes)
		if err != nil {
			return nil, err
		}

		if headerLine == "" {
			break
		}

		name, value, ok := strings.Cut(headerLine, ":")
		if !ok {
			return nil, fmt.Errorf("malformed header: %q", headerLine)
		}

		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" {
			return nil, fmt.Errorf("empty header name")
		}

		req.headers[textproto.CanonicalMIMEHeaderKey(name)] = value
	}

	contentLength := 0
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
		if _, err := io.ReadFull(reader, req.body); err != nil {
			return nil, fmt.Errorf("incomplete request body: %w", err)
		}
	}

	return req, nil
}

func readLine(reader *bufio.Reader, maxBytes int) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF && len(line) > 0 {
			return "", fmt.Errorf("incomplete line")
		}
		if err == io.EOF {
			return "", fmt.Errorf("unexpected end of request")
		}
		return "", err
	}

	if len(line) > maxBytes {
		return "", fmt.Errorf("line exceeds %d bytes", maxBytes)
	}

	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}
