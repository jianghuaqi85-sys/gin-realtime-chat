package main

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

func main() {
	baseURL := "http://localhost:8080"

	fmt.Println("=== Performance Benchmark ===")
	fmt.Println("Target:", baseURL)
	fmt.Println()

	healthCheck(baseURL)

	concurrency := []int{10, 50, 100, 200, 500}
	duration := 10 * time.Second

	for _, concur := range concurrency {
		fmt.Printf("Testing with %d concurrent requests...\n", concur)
		results := runBenchmark(baseURL+"/api/public/health", concur, duration)
		fmt.Printf("  QPS: %.2f, Avg Latency: %.2fms, P99: %.2fms\n",
			results.qps, results.avgLatency, results.p99Latency)
		fmt.Println()
	}
}

func healthCheck(url string) {
	resp, err := http.Get(url + "/api/public/health")
	if err != nil {
		fmt.Println("Health check failed:", err)
		return
	}
	defer resp.Body.Close()
	fmt.Println("Health check passed")
}

type BenchmarkResult struct {
	qps        float64
	avgLatency float64
	p99Latency float64
}

func runBenchmark(url string, concurrency int, duration time.Duration) BenchmarkResult {
	var wg sync.WaitGroup
	latencies := make([]float64, 0, concurrency*100)
	var mu sync.Mutex
	start := time.Now()
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					reqStart := time.Now()
					resp, err := http.Get(url)
					latency := time.Since(reqStart).Seconds() * 1000

					if err != nil {
						continue
					}
					resp.Body.Close()

					mu.Lock()
					latencies = append(latencies, latency)
					mu.Unlock()
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)
	wg.Wait()

	elapsed := time.Since(start).Seconds()
	qps := float64(len(latencies)) / elapsed

	sum := 0.0
	for _, l := range latencies {
		sum += l
	}
	avgLatency := sum / float64(len(latencies))

	sort.Float64s(latencies)
	p99Index := int(float64(len(latencies)) * 0.99)
	p99Latency := latencies[p99Index]

	return BenchmarkResult{
		qps:        qps,
		avgLatency: avgLatency,
		p99Latency: p99Latency,
	}
}
