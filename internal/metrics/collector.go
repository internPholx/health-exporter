package metrics

import (
	"fmt"
	"strings"
	"sync"
)

type Collector struct {
	mu     sync.Mutex
	status map[string]float64
}

func New() *Collector {
	return &Collector{
		status: make(map[string]float64),
	}
}

func (c *Collector) Set(url string, up bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if up {
		c.status[url] = 1.0
	} else {
		c.status[url] = 0.0
	}
}

func (c *Collector) Render() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("# HELP service_up Is the service up (1) or down (0)\n")
	sb.WriteString("# TYPE service_up gauge\n")

	for url, val := range c.status {
		fmt.Fprintf(&sb, "service_up{url=%q} %g\n", url, val)
	}
	return sb.String()
}
