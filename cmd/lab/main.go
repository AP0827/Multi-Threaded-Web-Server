package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "compare":
		if err := runCompare(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "compare error:", err)
			os.Exit(1)
		}
	case "benchmark":
		if err := runBenchmark(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "benchmark error:", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/lab compare -mtws 127.0.0.1:8080 -proxy 127.0.0.1:8081")
	fmt.Println("  go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 200 -concurrency 10")
}

func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	mtwsAddr := fs.String("mtws", "127.0.0.1:8080", "direct MTWS TCP address")
	proxyAddr := fs.String("proxy", "127.0.0.1:8081", "ModSecurity proxy TCP address")
	payloadDir := fs.String("payload-dir", "experiments/payloads", "directory containing raw HTTP payload files")
	timeout := fs.Duration("timeout", 3*time.Second, "per-request timeout")
	jsonOut := fs.String("json-out", "", "optional file path for JSON results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	payloads, err := loadPayloads(*payloadDir)
	if err != nil {
		return err
	}

	fmt.Println("Payload comparison")
	fmt.Printf("%-28s %-18s %-18s %-12s\n", "Payload", "MTWS", "Proxy", "Divergence")

	report := compareReport{
		GeneratedAt: time.Now().UTC(),
		MTWSAddr:    *mtwsAddr,
		ProxyAddr:   *proxyAddr,
		PayloadDir:  *payloadDir,
	}

	for _, payload := range payloads {
		mtwsResult := executeRawRequest(*mtwsAddr, payload.Body, *timeout)
		proxyResult := executeRawRequest(*proxyAddr, payload.Body, *timeout)
		divergence := classifyDifference(mtwsResult, proxyResult)

		fmt.Printf(
			"%-28s %-18s %-18s %-12s\n",
			payload.Name,
			mtwsResult.Summary(),
			proxyResult.Summary(),
			divergence,
		)

		report.Results = append(report.Results, compareEntry{
			Payload:    payload.Name,
			MTWS:       mtwsResult.toSerializable(),
			Proxy:      proxyResult.toSerializable(),
			Divergence: divergence,
		})
	}

	if *jsonOut != "" {
		if err := writeJSON(*jsonOut, report); err != nil {
			return err
		}
	}

	return nil
}

func runBenchmark(args []string) error {
	fs := flag.NewFlagSet("benchmark", flag.ContinueOnError)
	targetURL := fs.String("url", "http://127.0.0.1:8080/health", "benchmark target URL")
	requests := fs.Int("requests", 200, "total number of requests")
	concurrency := fs.Int("concurrency", 10, "number of concurrent workers")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout")
	jsonOut := fs.String("json-out", "", "optional file path for JSON results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *requests <= 0 {
		return errors.New("requests must be > 0")
	}
	if *concurrency <= 0 {
		return errors.New("concurrency must be > 0")
	}

	latencies := make([]time.Duration, 0, *requests)
	statusCounts := make(map[int]int)
	errCount := 0

	client := &http.Client{
		Timeout: *timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	start := time.Now()
	var mu sync.Mutex
	jobs := make(chan struct{}, *requests)
	var wg sync.WaitGroup

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				reqStart := time.Now()
				resp, err := client.Get(*targetURL)
				latency := time.Since(reqStart)

				mu.Lock()
				latencies = append(latencies, latency)
				if err != nil {
					errCount++
					mu.Unlock()
					continue
				}
				statusCounts[resp.StatusCode]++
				mu.Unlock()

				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()
	}

	for i := 0; i < *requests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)
	wg.Wait()

	elapsed := time.Since(start)
	summary := summarizeLatencies(latencies)

	fmt.Println("Benchmark")
	fmt.Printf("URL: %s\n", *targetURL)
	fmt.Printf("Requests: %d\n", *requests)
	fmt.Printf("Concurrency: %d\n", *concurrency)
	fmt.Printf("Elapsed: %s\n", elapsed)
	fmt.Printf("Throughput: %.2f req/s\n", float64(len(latencies))/elapsed.Seconds())
	fmt.Printf("Latency avg: %s\n", summary.Average)
	fmt.Printf("Latency min: %s\n", summary.Min)
	fmt.Printf("Latency p50: %s\n", summary.P50)
	fmt.Printf("Latency p95: %s\n", summary.P95)
	fmt.Printf("Latency p99: %s\n", summary.P99)
	fmt.Printf("Latency max: %s\n", summary.Max)
	fmt.Printf("Transport errors: %d\n", errCount)

	if len(statusCounts) > 0 {
		fmt.Println("Status codes:")
		for _, code := range sortedStatusCodes(statusCounts) {
			fmt.Printf("  %d -> %d\n", code, statusCounts[code])
		}
	}

	if *jsonOut != "" {
		report := benchmarkReport{
			GeneratedAt: time.Now().UTC(),
			URL:         *targetURL,
			Requests:    *requests,
			Concurrency: *concurrency,
			Elapsed:     elapsed.String(),
			Throughput:  float64(len(latencies)) / elapsed.Seconds(),
			Errors:      errCount,
			StatusCodes: statusCounts,
			Latency: latencyStats{
				Average: summary.Average.String(),
				Min:     summary.Min.String(),
				P50:     summary.P50.String(),
				P95:     summary.P95.String(),
				P99:     summary.P99.String(),
				Max:     summary.Max.String(),
			},
		}

		if err := writeJSON(*jsonOut, report); err != nil {
			return err
		}
	}

	return nil
}

type payloadCase struct {
	Name string
	Body []byte
}

func loadPayloads(dir string) ([]payloadCase, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	payloads := make([]payloadCase, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".http" {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}

		payloads = append(payloads, payloadCase{
			Name: entry.Name(),
			Body: body,
		})
	}

	sort.Slice(payloads, func(i, j int) bool {
		return payloads[i].Name < payloads[j].Name
	})

	return payloads, nil
}

type rawResult struct {
	StatusCode int
	StatusLine string
	Err        error
}

func (r rawResult) Summary() string {
	if r.Err != nil {
		return "error:" + r.Err.Error()
	}
	if r.StatusLine == "" {
		return "no-response"
	}
	return fmt.Sprintf("%d %s", r.StatusCode, r.StatusLine)
}

func executeRawRequest(addr string, request []byte, timeout time.Duration) rawResult {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return rawResult{Err: err}
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return rawResult{Err: err}
	}

	if _, err := conn.Write(request); err != nil {
		return rawResult{Err: err}
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		return rawResult{Err: err}
	}
	statusLine = strings.TrimSpace(statusLine)

	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		return rawResult{StatusLine: statusLine}
	}

	var statusCode int
	_, scanErr := fmt.Sscanf(parts[1], "%d", &statusCode)
	if scanErr != nil {
		return rawResult{StatusLine: statusLine}
	}

	return rawResult{
		StatusCode: statusCode,
		StatusLine: statusLine,
	}
}

func (r rawResult) toSerializable() rawResultJSON {
	result := rawResultJSON{
		StatusCode: r.StatusCode,
		StatusLine: r.StatusLine,
	}
	if r.Err != nil {
		result.Error = r.Err.Error()
	}
	return result
}

func classifyDifference(left rawResult, right rawResult) string {
	if left.Err != nil && right.Err != nil {
		return "unavailable"
	}
	if (left.Err != nil) != (right.Err != nil) {
		return "infra-gap"
	}
	if left.StatusCode != right.StatusCode {
		return "yes"
	}
	return "no"
}

type latencySummary struct {
	Average time.Duration
	Min     time.Duration
	P50     time.Duration
	P95     time.Duration
	P99     time.Duration
	Max     time.Duration
}

func summarizeLatencies(samples []time.Duration) latencySummary {
	if len(samples) == 0 {
		return latencySummary{}
	}

	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	var total time.Duration
	for _, sample := range sorted {
		total += sample
	}

	return latencySummary{
		Average: time.Duration(int64(total) / int64(len(sorted))),
		Min:     sorted[0],
		P50:     percentile(sorted, 0.50),
		P95:     percentile(sorted, 0.95),
		P99:     percentile(sorted, 0.99),
		Max:     sorted[len(sorted)-1],
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	index := int(math.Ceil((float64(len(sorted)) * p))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func sortedStatusCodes(counts map[int]int) []int {
	codes := make([]int, 0, len(counts))
	for code := range counts {
		codes = append(codes, code)
	}
	sort.Ints(codes)
	return codes
}

type compareReport struct {
	GeneratedAt time.Time      `json:"generated_at"`
	MTWSAddr    string         `json:"mtws_addr"`
	ProxyAddr   string         `json:"proxy_addr"`
	PayloadDir  string         `json:"payload_dir"`
	Results     []compareEntry `json:"results"`
}

type compareEntry struct {
	Payload    string        `json:"payload"`
	MTWS       rawResultJSON `json:"mtws"`
	Proxy      rawResultJSON `json:"proxy"`
	Divergence string        `json:"divergence"`
}

type rawResultJSON struct {
	StatusCode int    `json:"status_code"`
	StatusLine string `json:"status_line"`
	Error      string `json:"error,omitempty"`
}

type benchmarkReport struct {
	GeneratedAt time.Time    `json:"generated_at"`
	URL         string       `json:"url"`
	Requests    int          `json:"requests"`
	Concurrency int          `json:"concurrency"`
	Elapsed     string       `json:"elapsed"`
	Throughput  float64      `json:"throughput"`
	Errors      int          `json:"errors"`
	StatusCodes map[int]int  `json:"status_codes"`
	Latency     latencyStats `json:"latency"`
}

type latencyStats struct {
	Average string `json:"average"`
	Min     string `json:"min"`
	P50     string `json:"p50"`
	P95     string `json:"p95"`
	P99     string `json:"p99"`
	Max     string `json:"max"`
}

func writeJSON(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
