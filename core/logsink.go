package core

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type LogStore struct {
	mu      sync.Mutex
	max     int
	entries []LogEntry
}

type LogWriter struct {
	store *LogStore
}

var defaultLogStore = NewLogStore(200)

func NewLogStore(max int) *LogStore {
	if max <= 0 {
		max = 200
	}
	return &LogStore{max: max, entries: make([]LogEntry, 0, max)}
}

func NewLogWriter() io.Writer {
	return &LogWriter{store: defaultLogStore}
}

func RecentLogs(limit int) []LogEntry {
	return defaultLogStore.Snapshot(limit)
}

func (s *LogStore) Add(message string) {
	message = strings.TrimSpace(message)
	if s == nil || message == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) >= s.max {
		copy(s.entries, s.entries[1:])
		s.entries[len(s.entries)-1] = LogEntry{Time: time.Now().UTC().Format(time.RFC3339Nano), Message: message}
		return
	}

	s.entries = append(s.entries, LogEntry{Time: time.Now().UTC().Format(time.RFC3339Nano), Message: message})
}

func (s *LogStore) Snapshot(limit int) []LogEntry {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > len(s.entries) {
		limit = len(s.entries)
	}

	start := len(s.entries) - limit
	result := make([]LogEntry, limit)
	copy(result, s.entries[start:])
	return result
}

func (w *LogWriter) Write(p []byte) (int, error) {
	if w == nil || w.store == nil {
		return len(p), nil
	}

	chunks := bytes.Split(bytes.TrimRight(p, "\n"), []byte("\n"))
	for _, chunk := range chunks {
		line := strings.TrimSpace(string(chunk))
		if line != "" {
			w.store.Add(line)
		}
	}

	return len(p), nil
}
