package main

import (
	"fmt"
	"io"
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
		fmt.Printf("  QPS: %.2f\n", results.qps)
		fmt.Printf("  Success: %d, Errors: %d\n", results.successCount, results.errorCount)
		if results.successCount > 0 {
			fmt.Printf("  Avg Latency: %.2fms, P99: %.2fms\n", results.avgLatency, results.p99Latency)
		}
		fmt.Println("--------------------------------------------------")
	}
}

func healthCheck(url string) {
	// 使用带超时的 Client，防止健康检查无限阻塞
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/api/public/health")
	if err != nil {
		fmt.Println("Health check failed:", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // 必须读取完 Body 才能复用连接
	fmt.Println("Health check passed")
}

type BenchmarkResult struct {
	qps          float64
	avgLatency   float64
	p99Latency   float64
	successCount int
	errorCount   int
}

func runBenchmark(url string, concurrency int, duration time.Duration) BenchmarkResult {
	// 1. 核心优化：定制连接池，匹配高并发需求
	tr := &http.Transport{
		MaxIdleConns:        concurrency,
		MaxIdleConnsPerHost: concurrency,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second, // 设置合理的超时时间
	}

	var wg sync.WaitGroup
	start := time.Now()
	done := make(chan struct{}) // 使用 struct{} 节省内存

	// 用于收集每个 Goroutine 局部结果的通道
	type workerResult struct {
		latencies []float64
		errs      int
	}
	resultsCh := make(chan workerResult, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// 2. 核心优化：无锁设计，每个 goroutine 维护自己的 slice
			localLatencies := make([]float64, 0, 1000)
			localErrors := 0

			for {
				select {
				case <-done:
					// 压测结束，上报当前 worker 的数据
					resultsCh <- workerResult{
						latencies: localLatencies,
						errs:      localErrors,
					}
					return
				default:
					reqStart := time.Now()
					resp, err := client.Get(url)

					if err != nil {
						localErrors++
						continue
					}

					// 必须读取完并关闭 Body，否则连接不会回到连接池
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()

					latency := time.Since(reqStart).Seconds() * 1000
					localLatencies = append(localLatencies, latency)
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)
	wg.Wait()
	close(resultsCh)

	elapsed := time.Since(start).Seconds()

	// 3. 数据聚合
	var allLatencies []float64
	totalErrors := 0
	for res := range resultsCh {
		allLatencies = append(allLatencies, res.latencies...)
		totalErrors += res.errs
	}

	successCount := len(allLatencies)
	qps := float64(successCount+totalErrors) / elapsed // QPS 通常包含所有发出的请求

	if successCount == 0 {
		return BenchmarkResult{
			qps:          qps,
			successCount: 0,
			errorCount:   totalErrors,
		}
	}

	// 4. 计算指标
	sum := 0.0
	for _, l := range allLatencies {
		sum += l
	}
	avgLatency := sum / float64(successCount)

	sort.Float64s(allLatencies)
	p99Index := int(float64(successCount) * 0.99)
	if p99Index >= successCount { // 防御性保护
		p99Index = successCount - 1
	}
	p99Latency := allLatencies[p99Index]

	return BenchmarkResult{
		qps:          qps,
		avgLatency:   avgLatency,
		p99Latency:   p99Latency,
		successCount: successCount,
		errorCount:   totalErrors,
	}
}
