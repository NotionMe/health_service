package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type CheckResult struct {
	URL        string
	StatusCode int
	Duration   time.Duration
	Error      error
}

type Health interface {
	CheckHealth(url string) CheckResult
}

type HTTPChecker struct {
	client *http.Client
}

func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		client: &http.Client{Timeout: timeout},
	}
}

func (c *HTTPChecker) CheckHealth(url string) CheckResult {
	start := time.Now()
	resp, err := c.client.Get(url)
	duration := time.Since(start)

	if err != nil {
		return CheckResult{
			URL:      url,
			Duration: duration,
			Error:    err,
		}
	}
	defer resp.Body.Close()

	return CheckResult{
		URL:        url,
		StatusCode: resp.StatusCode,
		Duration:   duration,
	}
}

func main() {
	urls := []string{
		"https://go.dev",
		"https://github.com",
		"https://httpbin.org/status/404",
		"https://invalid-fake-host-test.org",
	}

	checker := NewHTTPChecker(5 * time.Second)

	seqStart := time.Now()
	for _, u := range urls {
		checker.CheckHealth(u)
	}
	seqDuration := time.Since(seqStart)

	fmt.Println("=== HTTP URL Health Checker ===")
	fmt.Println("[Конкурентна перевірка адрес...]")

	parStart := time.Now()
	results := make(chan CheckResult, len(urls))
	var wg sync.WaitGroup

	for _, u := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			results <- checker.CheckHealth(url)
		}(u)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var success, fail int
	for res := range results {
		if res.Error != nil {
			fail++
			fmt.Printf("-> %s: Помилка (%v)\n", res.URL, res.Error)
		} else {
			success++
			fmt.Printf("-> %s: Status %d %s (час: %v)\n",
				res.URL,
				res.StatusCode,
				http.StatusText(res.StatusCode),
				res.Duration.Round(time.Millisecond),
			)
		}
	}
	parDuration := time.Since(parStart)

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("ПІДСУМКОВИЙ ЗВІТ МОНІТОРИНГУ:")
	fmt.Printf("- Усього перевірено адрес: %d\n", len(urls))
	fmt.Printf("- Успішних з'єднань: %d\n", success)
	fmt.Printf("- Мережевих помилок: %d\n", fail)
	fmt.Printf("- Загальний час роботи: %v (проти %v у послідовному режимі)\n",
		parDuration.Round(time.Millisecond),
		seqDuration.Round(time.Millisecond),
	)
	fmt.Println("==========================================")
}
