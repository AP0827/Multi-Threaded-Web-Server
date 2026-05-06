package waf

import "testing"

func TestScannerMatchesAcrossFeeds(t *testing.T) {
	scanner := New([]string{"union select"}).NewScanner()

	if match := scanner.Feed("uri", []byte("/search?q=union ")); match != nil {
		t.Fatalf("unexpected early match: %+v", match)
	}

	match := scanner.Feed("uri", []byte("select"))
	if match == nil {
		t.Fatal("expected a match")
	}
	if match.Pattern != "union select" {
		t.Fatalf("expected union select, got %q", match.Pattern)
	}
	if match.Field != "uri" {
		t.Fatalf("expected uri field, got %q", match.Field)
	}
}

func TestScannerIsCaseInsensitive(t *testing.T) {
	scanner := NewDefaultAutomaton().NewScanner()

	match := scanner.Feed("header:X-Test", []byte("<ScRiPt>alert(1)</ScRiPt>"))
	if match == nil {
		t.Fatal("expected script tag match")
	}
	if match.Pattern != "<script" {
		t.Fatalf("expected <script pattern, got %q", match.Pattern)
	}
}

func TestScannerNormalizesSQLComments(t *testing.T) {
	scanner := New([]string{"union select"}).NewScanner()

	match := scanner.Feed("uri", []byte("UNION/**/SELECT"))
	if match == nil {
		t.Fatal("expected sql comment obfuscation match")
	}
	if match.Pattern != "union select" {
		t.Fatalf("expected union select, got %q", match.Pattern)
	}
	if match.RuleID == "" {
		t.Fatal("expected rule id")
	}
}

func TestNormalizeCollapsesWhitespaceAndComments(t *testing.T) {
	got := string(Normalize([]byte("UNION\t/** hidden */\nSELECT")))
	if got != "union select" {
		t.Fatalf("expected normalized union select, got %q", got)
	}
}

func TestScannerMatchesOverlappingPatterns(t *testing.T) {
	scanner := New([]string{"abcd", "bc"}).NewScanner()

	match := scanner.Feed("field", []byte("zabc"))
	if match == nil {
		t.Fatal("expected overlapping pattern match")
	}
	if match.Pattern != "bc" && match.Pattern != "abcd" {
		t.Fatalf("unexpected pattern %q", match.Pattern)
	}
}
