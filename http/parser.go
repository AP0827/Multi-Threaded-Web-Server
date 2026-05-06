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
	maxTrailerCount     = 16
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
		method:   method,
		path:     path,
		version:  version,
		headers:  make(map[string]string),
		trailers: make(map[string]string),
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
		if _, hasContentLength := req.headers["Content-Length"]; hasContentLength {
			return nil, fmt.Errorf("content-length is not allowed with transfer-encoding")
		}
		if !isChunkedTransferEncoding(transferEncoding) {
			return nil, fmt.Errorf("unsupported transfer-encoding: %q", transferEncoding)
		}
	}

	host, ok := req.headers["Host"]
	if !ok || strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("missing required Host header")
	}
	if strings.ContainsAny(host, " \t\r\n") {
		return nil, fmt.Errorf("invalid Host header")
	}

	if cl, ok := req.headers["Content-Length"]; ok {
		if !isDecimal(cl) {
			return nil, fmt.Errorf("invalid content-length: %q", cl)
		}
		parsedLen, err := strconv.Atoi(cl)
		if err != nil {
			return nil, fmt.Errorf("invalid content-length: %q", cl)
		}
		if parsedLen > maxBodyBytes {
			return nil, fmt.Errorf("request body too large")
		}
		contentLength = parsedLen
	}

	if transferEncoding, ok := req.headers["Transfer-Encoding"]; ok && isChunkedTransferEncoding(transferEncoding) {
		body, trailers, err := readChunkedBody(reader, automaton)
		if err != nil {
			return nil, err
		}
		req.body = body
		req.trailers = trailers
	} else if contentLength > 0 {
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
	if !isToken(parts[0]) {
		return "", "", "", fmt.Errorf("invalid method: %q", parts[0])
	}
	if !isOriginForm(parts[0], parts[1]) {
		return "", "", "", fmt.Errorf("unsupported request target: %q", parts[1])
	}

	if match := scanField(automaton, "uri", []byte(parts[1])); match != nil {
		return "", "", "", securityError(match)
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
	if !isToken(name) {
		return "", "", false, fmt.Errorf("invalid header name: %q", name)
	}

	canonicalName := textproto.CanonicalMIMEHeaderKey(name)
	fieldName := "header:" + canonicalName
	if match := scanField(automaton, fieldName, []byte(value)); match != nil {
		return "", "", false, securityError(match)
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

	scanner := automaton.NewScanner()
	if match := scanner.Feed(field, value); match != nil {
		return match
	}
	return scanner.Finish(field)
}

func readBody(reader *bufio.Reader, body []byte, automaton *waf.Automaton) error {
	if len(body) == 0 {
		return nil
	}

	var scanner *waf.Scanner
	if automaton != nil {
		scanner = automaton.NewScanner()
	}
	offset := 0
	for offset < len(body) {
		n, err := reader.Read(body[offset:])
		if n > 0 {
			if scanner != nil {
				if match := scanner.Feed("body", body[offset:offset+n]); match != nil {
					return securityError(match)
				}
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
	if scanner != nil {
		if match := scanner.Finish("body"); match != nil {
			return securityError(match)
		}
	}

	return nil
}

func readChunkedBody(reader *bufio.Reader, automaton *waf.Automaton) ([]byte, map[string]string, error) {
	var body []byte
	scanner := newScanner(automaton)

	for {
		sizeLine, err := readLineLimited(reader, maxHeaderLineBytes)
		if err != nil {
			return nil, nil, err
		}

		chunkSize, err := parseChunkSize(string(sizeLine))
		if err != nil {
			return nil, nil, err
		}
		if chunkSize == 0 {
			if scanner != nil {
				if match := scanner.Finish("body"); match != nil {
					return nil, nil, securityError(match)
				}
			}
			trailers, err := readTrailers(reader, automaton)
			if err != nil {
				return nil, nil, err
			}
			return body, trailers, nil
		}
		if len(body)+chunkSize > maxBodyBytes {
			return nil, nil, fmt.Errorf("request body too large")
		}

		start := len(body)
		body = append(body, make([]byte, chunkSize)...)
		if _, err := io.ReadFull(reader, body[start:]); err != nil {
			return nil, nil, fmt.Errorf("incomplete chunk data: %w", err)
		}
		if scanner != nil {
			if match := scanner.Feed("body", body[start:]); match != nil {
				return nil, nil, securityError(match)
			}
		}

		if err := readCRLF(reader); err != nil {
			return nil, nil, err
		}
	}
}

func readTrailers(reader *bufio.Reader, automaton *waf.Automaton) (map[string]string, error) {
	trailers := make(map[string]string)
	seen := make(map[string]struct{})

	for i := 0; ; i++ {
		if i >= maxTrailerCount {
			return nil, fmt.Errorf("too many trailers")
		}

		name, value, done, err := readHeaderLine(reader, automaton)
		if err != nil {
			return nil, err
		}
		if done {
			return trailers, nil
		}
		if isForbiddenTrailer(name) {
			return nil, fmt.Errorf("forbidden trailer field: %s", name)
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate trailer: %s", name)
		}
		seen[name] = struct{}{}
		trailers[name] = value
	}
}

func parseChunkSize(line string) (int, error) {
	sizeText := line
	if value, _, ok := strings.Cut(line, ";"); ok {
		sizeText = value
	}
	sizeText = strings.TrimSpace(sizeText)
	if sizeText == "" {
		return 0, fmt.Errorf("empty chunk size")
	}

	size := 0
	for i := 0; i < len(sizeText); i++ {
		value, ok := hexValue(sizeText[i])
		if !ok {
			return 0, fmt.Errorf("invalid chunk size: %q", line)
		}
		if size > (maxBodyBytes-value)/16 {
			return 0, fmt.Errorf("chunk size too large")
		}
		size = size*16 + value
	}

	return size, nil
}

func readCRLF(reader *bufio.Reader) error {
	first, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("incomplete chunk terminator: %w", err)
	}
	second, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("incomplete chunk terminator: %w", err)
	}
	if first != '\r' || second != '\n' {
		return fmt.Errorf("invalid chunk terminator")
	}
	return nil
}

func newScanner(automaton *waf.Automaton) *waf.Scanner {
	if automaton == nil {
		return nil
	}
	return automaton.NewScanner()
}

func isChunkedTransferEncoding(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "chunked")
}

func isForbiddenTrailer(name string) bool {
	switch name {
	case "Content-Length", "Host", "Transfer-Encoding", "Trailer":
		return true
	default:
		return false
	}
}

func hexValue(b byte) (int, bool) {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0'), true
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10, true
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10, true
	default:
		return 0, false
	}
}

func securityError(match *waf.Match) *SecurityError {
	return &SecurityError{
		Pattern: match.Pattern,
		Field:   match.Field,
		RuleID:  match.RuleID,
	}
}

func isOriginForm(method string, target string) bool {
	if target == "*" {
		return method == "OPTIONS"
	}
	if !strings.HasPrefix(target, "/") {
		return false
	}
	for i := 0; i < len(target); i++ {
		if target[i] <= 31 || target[i] == 127 {
			return false
		}
	}
	return true
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func isToken(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		if !isTokenChar(value[i]) {
			return false
		}
	}
	return true
}

func isTokenChar(b byte) bool {
	if b >= 'A' && b <= 'Z' {
		return true
	}
	if b >= 'a' && b <= 'z' {
		return true
	}
	if b >= '0' && b <= '9' {
		return true
	}

	switch b {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
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
				return nil, io.EOF
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
