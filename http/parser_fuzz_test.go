package mtwshttp

import (
	"bufio"
	"strings"
	"testing"
)

func FuzzParseRequestNeverPanics(f *testing.F) {
	f.Add("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	f.Add("POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\n\r\nhello")
	f.Add("POST / HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n5\r\nhello\r\n0\r\n\r\n")
	f.Add("GET /?q=UNION%20SELECT HTTP/1.1\r\nHost: localhost\r\n\r\n")

	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = ParseRequest(bufio.NewReader(strings.NewReader(raw)))
	})
}
