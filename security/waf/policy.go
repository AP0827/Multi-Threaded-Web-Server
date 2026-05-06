package waf

import (
	"bufio"
	"log"
	"os"
	"strings"
)

const policyFileEnv = "MTWS_WAF_POLICY_FILE"

var defaultPatterns = []string{
	"union select",
	"select * from",
	"information_schema",
	"drop table",
	"sleep(",
	"benchmark(",
	"' or 1=1",
	`" or 1=1`,
	"<script",
	"</script",
	"javascript:",
	"onerror=",
	"onload=",
	"../",
	"..\\",
	"/etc/passwd",
	"cmd.exe",
	"/bin/sh",
	"powershell -",
	"169.254.169.254",
	"metadata.google.internal",
}

func DefaultPatterns() []string {
	patterns := make([]string, len(defaultPatterns))
	copy(patterns, defaultPatterns)
	return patterns
}

func NewDefaultAutomaton() *Automaton {
	return New(LoadPatterns())
}

func LoadPatterns() []string {
	policyFile := strings.TrimSpace(os.Getenv(policyFileEnv))
	if policyFile == "" {
		return DefaultPatterns()
	}

	patterns, err := LoadPatternsFile(policyFile)
	if err != nil {
		log.Printf("Failed to load WAF policy file %q: %v; using defaults", policyFile, err)
		return DefaultPatterns()
	}
	if len(patterns) == 0 {
		log.Printf("WAF policy file %q did not contain patterns; using defaults", policyFile)
		return DefaultPatterns()
	}

	return patterns
}

func LoadPatternsFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return patterns, nil
}
