package main

import (
	"bufio"
	"context"
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
	case "summarize":
		if err := runSummarize(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "summarize error:", err)
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
	fmt.Println("  go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -duration 2m -concurrency 10 -keepalive")
	fmt.Println("  go run ./cmd/lab summarize -glob \"experiments/results/benchmark-*.json\"")
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
	duration := fs.Duration("duration", 0, "optional time-boxed soak duration; when set, requests is ignored")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout")
	keepAlive := fs.Bool("keepalive", false, "reuse HTTP connections during benchmarking")
	jsonOut := fs.String("json-out", "", "optional file path for JSON results")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *duration <= 0 && *requests <= 0 {
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
			DisableKeepAlives: !*keepAlive,
		},
	}

	start := time.Now()
	var mu sync.Mutex
	var wg sync.WaitGroup

	record := func(latency time.Duration, statusCode int, err error) {
		mu.Lock()
		defer mu.Unlock()
		latencies = append(latencies, latency)
		if err != nil {
			errCount++
			return
		}
		statusCounts[statusCode]++
	}

	issue := func() {
		reqStart := time.Now()
		resp, err := client.Get(*targetURL)
		latency := time.Since(reqStart)
		if err != nil {
			record(latency, 0, err)
			return
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		record(latency, resp.StatusCode, nil)
	}

	if *duration > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), *duration)
		defer cancel()

		for i := 0; i < *concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						issue()
					}
				}
			}()
		}
	} else {
		jobs := make(chan struct{}, *requests)
		for i := 0; i < *concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range jobs {
					issue()
				}
			}()
		}

		for i := 0; i < *requests; i++ {
			jobs <- struct{}{}
		}
		close(jobs)
	}

	wg.Wait()

	completedRequests := len(latencies)
	configuredRequests := *requests
	if *duration > 0 {
		configuredRequests = completedRequests
	}

	elapsed := time.Since(start)
	summary := summarizeLatencies(latencies)
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(completedRequests) / elapsed.Seconds()
	}

	fmt.Println("Benchmark")
	fmt.Printf("URL: %s\n", *targetURL)
	if *duration > 0 {
		fmt.Println("Mode: soak")
		fmt.Printf("Duration target: %s\n", *duration)
	} else {
		fmt.Println("Mode: fixed-requests")
	}
	fmt.Printf("Requests: %d\n", completedRequests)
	fmt.Printf("Concurrency: %d\n", *concurrency)
	fmt.Printf("Keep-alive: %t\n", *keepAlive)
	fmt.Printf("Elapsed: %s\n", elapsed)
	fmt.Printf("Throughput: %.2f req/s\n", throughput)
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
	if statusCounts[http.StatusTooManyRequests] > 0 {
		fmt.Println("Warning: 429 responses observed; do not use this run as clean parser/WAF latency evidence.")
	}

	if *jsonOut != "" {
		report := benchmarkReport{
			GeneratedAt: time.Now().UTC(),
			URL:         *targetURL,
			Requests:    configuredRequests,
			Concurrency: *concurrency,
			Duration:    durationString(*duration),
			KeepAlive:   *keepAlive,
			Elapsed:     elapsed.String(),
			Throughput:  throughput,
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

func runSummarize(args []string) error {
	fs := flag.NewFlagSet("summarize", flag.ContinueOnError)
	glob := fs.String("glob", "", "glob pattern for benchmark JSON files")
	if err := fs.Parse(args); err != nil {
		return err
	}

	files := fs.Args()
	if *glob != "" {
		matches, err := filepath.Glob(*glob)
		if err != nil {
			return err
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		return errors.New("provide at least one benchmark JSON file or -glob pattern")
	}

	reportsByURL := make(map[string][]benchmarkReport)
	for _, file := range files {
		report, err := readBenchmarkReport(file)
		if err != nil {
			return err
		}
		reportsByURL[report.URL] = append(reportsByURL[report.URL], report)
	}

	fmt.Println("Benchmark summary")
	fmt.Printf("%-34s %-5s %-12s %-12s %-12s %-12s %-12s\n", "URL", "Runs", "Median RPS", "Median Avg", "Median P50", "Median P95", "Median P99")
	for _, url := range sortedURLs(reportsByURL) {
		summary, err := summarizeBenchmarkReports(reportsByURL[url])
		if err != nil {
			return err
		}
		fmt.Printf(
			"%-34s %-5d %-12.2f %-12s %-12s %-12s %-12s\n",
			url,
			summary.Runs,
			summary.Throughput,
			summary.Average,
			summary.P50,
			summary.P95,
			summary.P99,
		)
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
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".http" && ext != ".raw" {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if ext == ".http" {
			body = normalizeHTTPFixture(body)
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

func normalizeHTTPFixture(body []byte) []byte {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\n", "\r\n")
	if !strings.HasSuffix(text, "\r\n\r\n") {
		text = strings.TrimRight(text, "\r\n") + "\r\n\r\n"
	}
	return []byte(text)
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

type benchmarkSummary struct {
	Runs       int
	Throughput float64
	Average    time.Duration
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
}

func readBenchmarkReport(path string) (benchmarkReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return benchmarkReport{}, err
	}

	var report benchmarkReport
	if err := json.Unmarshal(data, &report); err != nil {
		return benchmarkReport{}, err
	}
	if report.URL == "" {
		return benchmarkReport{}, fmt.Errorf("%s is not a benchmark report", path)
	}

	return report, nil
}

func summarizeBenchmarkReports(reports []benchmarkReport) (benchmarkSummary, error) {
	if len(reports) == 0 {
		return benchmarkSummary{}, errors.New("no benchmark reports")
	}

	throughputs := make([]float64, 0, len(reports))
	averages := make([]time.Duration, 0, len(reports))
	p50s := make([]time.Duration, 0, len(reports))
	p95s := make([]time.Duration, 0, len(reports))
	p99s := make([]time.Duration, 0, len(reports))

	for _, report := range reports {
		average, err := time.ParseDuration(report.Latency.Average)
		if err != nil {
			return benchmarkSummary{}, err
		}
		p50, err := time.ParseDuration(report.Latency.P50)
		if err != nil {
			return benchmarkSummary{}, err
		}
		p95, err := time.ParseDuration(report.Latency.P95)
		if err != nil {
			return benchmarkSummary{}, err
		}
		p99, err := time.ParseDuration(report.Latency.P99)
		if err != nil {
			return benchmarkSummary{}, err
		}

		throughputs = append(throughputs, report.Throughput)
		averages = append(averages, average)
		p50s = append(p50s, p50)
		p95s = append(p95s, p95)
		p99s = append(p99s, p99)
	}

	return benchmarkSummary{
		Runs:       len(reports),
		Throughput: medianFloat64(throughputs),
		Average:    medianDuration(averages),
		P50:        medianDuration(p50s),
		P95:        medianDuration(p95s),
		P99:        medianDuration(p99s),
	}, nil
}

func sortedURLs(reportsByURL map[string][]benchmarkReport) []string {
	urls := make([]string, 0, len(reportsByURL))
	for url := range reportsByURL {
		urls = append(urls, url)
	}
	sort.Strings(urls)
	return urls
}

func medianFloat64(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func medianDuration(values []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
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
	Duration    string       `json:"duration,omitempty"`
	KeepAlive   bool         `json:"keep_alive"`
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

func durationString(duration time.Duration) string {
	if duration <= 0 {
		return ""
	}
	return duration.String()
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
