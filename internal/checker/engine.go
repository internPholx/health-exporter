package checker

import (
    "context"
    "net/http"
    "sync"
    "time"

    "health-exporter/internal/models"
)

type Result struct {
    URL string
    Up  bool
}

func Run(cfg models.Config, concurrency int) []Result {
    jobs    := make(chan models.Service, len(cfg.Services))
    results := make(chan Result, len(cfg.Services))

    var wg sync.WaitGroup

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for svc := range jobs {
                up := checkWithRetry(svc.URL, 3)
                results <- Result{URL: svc.URL, Up: up}
            }
        }()
    }

    for _, svc := range cfg.Services {
        jobs <- svc
    }
    close(jobs)

    go func() {
        wg.Wait()
        close(results)
    }()

    var all []Result
    for r := range results {
        all = append(all, r)
    }
    return all
}

func checkWithRetry(url string, maxRetry int) bool {
    for i := 0; i < maxRetry; i++ {
        if checkURL(url) {
            return true
        }
        if i < maxRetry-1 {
            time.Sleep(500 * time.Millisecond)
        }
    }
    return false
}

func checkURL(url string) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return false
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode < 500
}
