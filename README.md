# health-exporter

A lightweight HTTP health checker that exposes service status as Prometheus metrics.

Periodically checks a list of URLs for availability and serves the results at `/metrics` in Prometheus text format.

```
service_up{url="https://www.google.com"} 1
service_up{url="https://my-api.internal"} 0
```

---

## Project Structure

```
health-exporter/
├── cmd/exporter/main.go          # Entry point, HTTP server, graceful shutdown
├── internal/
│   ├── checker/engine.go         # Worker pool, HTTP probes, retry logic
│   ├── metrics/collector.go      # Thread-safe result cache
│   └── models/config.go          # Config structs
├── services.yaml                 # List of URLs to monitor
└── prometheus.yml                # Example Prometheus scrape config
```

---

## Configuration

Edit `services.yaml` to define the services to monitor:

```yaml
services:
  - name: "Google"
    url: "https://www.google.com"
  - name: "My API"
    url: "https://my-api.internal/health"
  - name: "Cloudflare DNS"
    url: "https://1.1.1.1"
```

| Behaviour | Value |
|---|---|
| Check interval | 30 seconds |
| HTTP timeout per request | 2 seconds |
| Retries before marking DOWN | 3 |
| Worker concurrency | 5 |
| Metrics port | 9090 |

---

## Run Locally

```bash
# Install dependencies
go mod tidy

# Run
go run cmd/exporter/main.go

# Test
curl http://localhost:9090/metrics
```

---

## Deploy on Ubuntu Server (Proxmox)

### 1. Build a Linux binary on your Mac

```bash
GOOS=linux GOARCH=amd64 go build -o health-exporter ./cmd/exporter
```

### 2. Copy to server

```bash
scp health-exporter services.yaml user@<SERVER_IP>:/opt/health-exporter/
```

### 3. Create a systemd service

SSH into the server, then:

```bash
sudo nano /etc/systemd/system/health-exporter.service
```

Paste the following:

```ini
[Unit]
Description=Health Exporter
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/health-exporter
ExecStart=/opt/health-exporter/health-exporter
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable health-exporter
sudo systemctl start health-exporter

# Check status
sudo systemctl status health-exporter

# View logs
sudo journalctl -u health-exporter -f
```

### 4. Configure Prometheus

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: "health-exporter"
    static_configs:
      - targets: ["<SERVER_IP>:9090"]
```

Reload Prometheus:

```bash
sudo systemctl reload prometheus
```

---

## Metrics Reference

| Metric | Type | Description |
|---|---|---|
| `service_up` | Gauge | `1` = UP, `0` = DOWN |

Label: `url` — the full URL of the checked service.

---

## Tech Stack

- **Go** 1.26
- Standard library only (`net/http`, `sync`, `context`, `os/signal`)
- [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) for config parsing
