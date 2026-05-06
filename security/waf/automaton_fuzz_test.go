package waf

import (
	"strings"
	"testing"
)

func FuzzScannerMatchesReference(f *testing.F) {
	f.Add("union select", "/search?q=UNION SELECT")
	f.Add("<script", "<ScRiPt>alert(1)</script>")
	f.Add("../", "/../../etc/passwd")

	f.Fuzz(func(t *testing.T, pattern string, input string) {
		if pattern == "" || !isASCII(pattern) || !isASCII(input) {
			t.Skip()
		}
		normalizedPattern := Normalize([]byte(pattern))
		if len(normalizedPattern) == 0 {
			t.Skip()
		}

		scanner := New([]string{pattern}).NewScanner()
		match := scanner.Feed("field", []byte(input))
		if match == nil {
			match = scanner.Finish("field")
		}
		reference := strings.Contains(string(Normalize([]byte(input))), string(normalizedPattern))
		if (match != nil) != reference {
			t.Fatalf("scanner=%t reference=%t pattern=%q input=%q", match != nil, reference, pattern, input)
		}
	})
}

func isASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] > 127 {
			return false
		}
	}
	return true
}
