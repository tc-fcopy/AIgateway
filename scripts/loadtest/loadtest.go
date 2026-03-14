package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type headerList []string

func (h *headerList) String() string {
	return strings.Join(*h, ",")
}

func (h *headerList) Set(v string) error {
	*h = append(*h, v)
	return nil
}

type workerResult struct {
	requests    uint64
	errors      uint64
	status2xx   uint64
	status3xx   uint64
	status4xx   uint64
	status5xx   uint64
	statusOther uint64
	bytesIn     uint64
	latencyNs   []int64
}

func main() {
	var (
		targetURL   = flag.String("url", "", "Target URL, e.g. http://127.0.0.1:8080/ping")
		method      = flag.String("method", "GET", "HTTP method")
		concurrency = flag.Int("c", 50, "Number of workers")
		duration    = flag.Duration("d", 30*time.Second, "Test duration")
		timeout     = flag.Duration("timeout", 5*time.Second, "Request timeout")
		qps         = flag.Int("qps", 0, "Global QPS limit (0 = unlimited)")
		body        = flag.String("body", "", "Raw request body")
		bodyFile    = flag.String("body-file", "", "Path to request body file")
		warmup      = flag.Duration("warmup", 0, "Warmup duration (excluded from results)")
	)

	var headers headerList
	flag.Var(&headers, "H", "Request header, repeatable. Example: -H \"Authorization: Bearer xxx\"")
	flag.Parse()

	if *targetURL == "" {
		fmt.Fprintln(os.Stderr, "-url is required")
		os.Exit(2)
	}
	if *concurrency <= 0 {
		fmt.Fprintln(os.Stderr, "-c must be > 0")
		os.Exit(2)
	}
	if *duration <= 0 {
		fmt.Fprintln(os.Stderr, "-d must be > 0")
		os.Exit(2)
	}
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "-timeout must be > 0")
		os.Exit(2)
	}

	methodUpper := strings.ToUpper(strings.TrimSpace(*method))
	if methodUpper == "" {
		fmt.Fprintln(os.Stderr, "-method is invalid")
		os.Exit(2)
	}

	bodyBytes, err := loadBody(*body, *bodyFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	headerMap, err := parseHeaders(headers)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	transport := &http.Transport{
		MaxIdleConns:        *concurrency * 4,
		MaxIdleConnsPerHost: *concurrency * 4,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}
	client := &http.Client{
		Timeout:   *timeout,
		Transport: transport,
	}

	makeRequest := func() (*http.Request, error) {
		var bodyReader io.Reader
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(methodUpper, *targetURL, bodyReader)
		if err != nil {
			return nil, err
		}
		req.Header = headerMap.Clone()
		if len(bodyBytes) > 0 && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
		return req, nil
	}

	if *warmup > 0 {
		_ = runLoad(client, makeRequest, *concurrency, *warmup, *qps, false)
	}

	result := runLoad(client, makeRequest, *concurrency, *duration, *qps, true)
	printSummary(*targetURL, methodUpper, *concurrency, *duration, *timeout, *qps, result)
}

func loadBody(body string, bodyFile string) ([]byte, error) {
	if bodyFile != "" {
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return nil, fmt.Errorf("read body-file failed: %w", err)
		}
		return data, nil
	}
	if body != "" {
		return []byte(body), nil
	}
	return nil, nil
}

func parseHeaders(headers headerList) (http.Header, error) {
	headerMap := make(http.Header)
	for _, raw := range headers {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid header: %s", raw)
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			return nil, fmt.Errorf("invalid header: %s", raw)
		}
		headerMap.Add(name, value)
	}
	return headerMap, nil
}

type loadResult struct {
	duration    time.Duration
	requests    uint64
	errors      uint64
	status2xx   uint64
	status3xx   uint64
	status4xx   uint64
	status5xx   uint64
	statusOther uint64
	bytesIn     uint64
	latencies   []int64
}

func runLoad(client *http.Client, makeRequest func() (*http.Request, error), concurrency int, duration time.Duration, qps int, recordLatency bool) loadResult {
	start := time.Now()
	deadline := start.Add(duration)

	var ticker *time.Ticker
	var rateCh <-chan time.Time
	if qps > 0 {
		interval := time.Second / time.Duration(qps)
		if interval <= 0 {
			interval = time.Nanosecond
		}
		ticker = time.NewTicker(interval)
		rateCh = ticker.C
	}

	resultsCh := make(chan workerResult, concurrency)
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := workerResult{}

			for {
				if time.Now().After(deadline) {
					break
				}
				if rateCh != nil {
					<-rateCh
					if time.Now().After(deadline) {
						break
					}
				}

				req, err := makeRequest()
				if err != nil {
					res.requests++
					res.errors++
					continue
				}

				reqStart := time.Now()
				resp, err := client.Do(req)
				if err != nil {
					res.requests++
					res.errors++
					continue
				}

				n, _ := io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				res.requests++
				res.bytesIn += uint64(n)
				if recordLatency {
					res.latencyNs = append(res.latencyNs, time.Since(reqStart).Nanoseconds())
				}

				code := resp.StatusCode
				switch {
				case code >= 200 && code < 300:
					res.status2xx++
				case code >= 300 && code < 400:
					res.status3xx++
				case code >= 400 && code < 500:
					res.status4xx++
				case code >= 500 && code < 600:
					res.status5xx++
				default:
					res.statusOther++
				}
			}

			resultsCh <- res
		}()
	}

	wg.Wait()
	if ticker != nil {
		ticker.Stop()
	}
	close(resultsCh)

	result := loadResult{}
	result.duration = time.Since(start)

	for res := range resultsCh {
		result.requests += res.requests
		result.errors += res.errors
		result.status2xx += res.status2xx
		result.status3xx += res.status3xx
		result.status4xx += res.status4xx
		result.status5xx += res.status5xx
		result.statusOther += res.statusOther
		result.bytesIn += res.bytesIn
		if recordLatency {
			result.latencies = append(result.latencies, res.latencyNs...)
		}
	}

	return result
}

func printSummary(url string, method string, concurrency int, duration time.Duration, timeout time.Duration, qps int, result loadResult) {
	fmt.Println("Load test")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("Method: %s\n", method)
	fmt.Printf("Concurrency: %d\n", concurrency)
	fmt.Printf("Duration: %s\n", duration)
	fmt.Printf("Timeout: %s\n", timeout)
	if qps > 0 {
		fmt.Printf("QPS limit: %d\n", qps)
	} else {
		fmt.Println("QPS limit: unlimited")
	}

	fmt.Println("Results")
	fmt.Printf("Total requests: %d\n", result.requests)
	fmt.Printf("Errors: %d\n", result.errors)
	fmt.Printf("Status codes: 2xx=%d 3xx=%d 4xx=%d 5xx=%d other=%d\n", result.status2xx, result.status3xx, result.status4xx, result.status5xx, result.statusOther)

	latencyCount := len(result.latencies)
	if latencyCount > 0 {
		sort.Slice(result.latencies, func(i, j int) bool { return result.latencies[i] < result.latencies[j] })
		avg := averageNs(result.latencies)
		p50 := percentileNs(result.latencies, 50)
		p90 := percentileNs(result.latencies, 90)
		p99 := percentileNs(result.latencies, 99)
		max := result.latencies[latencyCount-1]
		fmt.Printf("Latency(ms): avg=%.2f p50=%.2f p90=%.2f p99=%.2f max=%.2f\n",
			nsToMs(avg), nsToMs(p50), nsToMs(p90), nsToMs(p99), nsToMs(max))
	} else {
		fmt.Println("Latency(ms): no samples")
	}

	seconds := result.duration.Seconds()
	if seconds <= 0 {
		seconds = duration.Seconds()
	}
	if seconds <= 0 {
		seconds = 1
	}
	rps := float64(result.requests) / seconds
	throughput := float64(result.bytesIn) / seconds / (1024 * 1024)

	fmt.Printf("RPS: %.2f\n", rps)
	fmt.Printf("Rx throughput: %.2f MB/s\n", throughput)
}

func averageNs(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return int64(float64(sum) / float64(len(values)))
}

func percentileNs(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := (p / 100.0) * float64(len(sorted))
	idx := int(math.Ceil(rank)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func nsToMs(ns int64) float64 {
	return float64(ns) / 1e6
}
