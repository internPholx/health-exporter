package scanner

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"health-exporter/internal/models"
)

// พอร์ตที่ HTTP service มักใช้
var commonPorts = []int{
	80, 443,
	3000, 4000, 5000,
	8000, 8080, 8081, 8443, 8888,
	9000, 9090, 9091, 9092, 9093,
}

// Run สแกน localhost และคืนรายชื่อ service ที่ตอบสนอง HTTP
func Run() []models.Service {
	found := make([]models.Service, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range commonPorts {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()

			// ลอง http ก่อน ถ้าไม่ได้ลอง https
			schemes := []string{"http", "https"}
			for _, scheme := range schemes {
				url := fmt.Sprintf("%s://localhost:%d", scheme, p)
				if probe(url) {
					mu.Lock()
					found = append(found, models.Service{
						Name: fmt.Sprintf("localhost:%d", p),
						URL:  url,
					})
					mu.Unlock()
					return // เจอแล้ว ไม่ต้องลอง scheme อื่น
				}
			}
		}(port)
	}

	wg.Wait()
	return found
}

// probe ลองยิง GET ไปที่ URL — ถ้ามีการตอบกลับ (ไม่ว่า status code อะไร) = มี service อยู่
func probe(url string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	// ปิด redirect เพื่อนับทุก response ว่า "มีอยู่"
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}
