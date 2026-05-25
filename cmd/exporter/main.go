package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"health-exporter/internal/checker"
	"health-exporter/internal/metrics"
	"health-exporter/internal/models"
	"health-exporter/internal/scanner"

	"gopkg.in/yaml.v3"
)

func main() {
	// ถ้าใส่ flag --scan จะสแกน localhost แล้วเขียนทับ services.yaml
	if len(os.Args) > 1 && os.Args[1] == "--scan" {
		runScan()
		return
	}

	data, err := os.ReadFile("services.yaml")
	if err != nil {
		log.Fatal("อ่านไฟล์ไม่ได้ (ลองรัน --scan ก่อน):", err)
	}
	var cfg models.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatal("parse yaml ไม่ได้:", err)
	}
	fmt.Printf("โหลด %d services\n", len(cfg.Services))

	col := metrics.New()

	// --- ตั้ง HTTP Server แบบที่ Shutdown ได้ ---
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, col.Render())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<a href="/metrics">metrics</a>`)
	})

	server := &http.Server{Addr: ":9090", Handler: mux}

	go runCheck(cfg, col)

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for range ticker.C {
			go runCheck(cfg, col)
		}
	}()

	go func() {
		fmt.Println("เปิด HTTP server ที่ :9090")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal("server error:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	// บอก OS ว่าให้ส่ง SIGINT (Ctrl+C) และ SIGTERM มาที่ channel นี้
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// บล็อก main ไว้รอ signal
	<-quit
	fmt.Println("\nกำลังปิดโปรแกรม...")

	// หยุด ticker ไม่ให้สั่งเช็คใหม่
	ticker.Stop()

	// Shutdown server อย่างสวยงาม — รอ request ที่ค้างอยู่ไม่เกิน 5 วิ
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Println("shutdown error:", err)
	}

	fmt.Println("ปิดเรียบร้อย")
}

func runCheck(cfg models.Config, col *metrics.Collector) {
	log.Println("กำลังเช็ค services...")
	results := checker.Run(cfg, 5)
	for _, r := range results {
		col.Set(r.URL, r.Up)
		status := "DOWN"
		if r.Up {
			status = "UP"
		}
		log.Printf("  %-45s %s", r.URL, status)
	}
}

// runScan สแกน localhost หา HTTP service แล้วเขียนผลลงใน services.yaml
func runScan() {
	fmt.Println("กำลังสแกน localhost...")
	services := scanner.Run()

	if len(services) == 0 {
		fmt.Println("ไม่พบ HTTP service บน localhost")
		return
	}

	fmt.Printf("พบ %d services:\n", len(services))
	for _, s := range services {
		fmt.Printf("  - %s (%s)\n", s.Name, s.URL)
	}

	cfg := models.Config{Services: services}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatal("marshal yaml ไม่ได้:", err)
	}

	if err := os.WriteFile("services.yaml", out, 0644); err != nil {
		log.Fatal("เขียนไฟล์ไม่ได้:", err)
	}

	fmt.Println("บันทึกลง services.yaml เรียบร้อย")
}
