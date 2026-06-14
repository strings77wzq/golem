# Monitoring Guide

Monitoring and observability for Golem: Prometheus metrics, Grafana dashboards, and alerting.

---

## Overview

Golem exposes Prometheus-compatible metrics at `/metrics` endpoint:

```
http://localhost:18790/metrics
```

Metrics are implemented in pure Go with no external dependencies.

---

## Quick Start

### 1. Start Monitoring Stack

```bash
cd docker/monitoring
docker compose up -d
```

Services:
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### 2. Verify Metrics

```bash
curl http://localhost:18790/metrics
```

---

## Available Metrics

<!-- AUTO-GENERATED from internal/metrics/ -->
### HTTP Metrics (Gateway)

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests |
| `http_request_duration_seconds` | Histogram | Request latency (buckets: 1ms to 10s) |
| `http_active_requests` | Gauge | In-flight requests |

### Extending Metrics

To add custom metrics, use the metrics registry:

```go
import "github.com/strings77wzq/golem/internal/metrics"

// Create custom metrics
myCounter := metrics.DefaultRegistry.NewCounter("my_counter", "Description")
myCounter.Inc()

myHistogram := metrics.DefaultRegistry.NewHistogram("my_histogram", "Description", []float64{0.1, 0.5, 1.0})
myHistogram.Observe(0.3)
```
<!-- END AUTO-GENERATED -->

---

## Prometheus Configuration

The included `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'golem'
    static_configs:
      - targets: ['host.docker.internal:18790']
    # Or for K8s:
    # - targets: ['golem-gateway:18790']
```

---

## Grafana Dashboard

An included dashboard (`grafana-dashboard.json`) provides:

### Dashboards

1. **Agent Overview**
   - Requests/sec
   - Error rate
   - Latency (p50, p95, p99)

2. **Token Usage**
   - Tokens per hour/day
   - Cost breakdown by model
   - Budget utilization

3. **Tool Performance**
   - Most used tools
   - Tool latency distribution

4. **Provider Health**
   - API response time
   - Error rate by provider
   - Rate limit hits

### Import Dashboard

1. Open Grafana: http://localhost:3000
2. Go to Dashboards → Import
3. Upload `grafana-dashboard.json`
4. Select Prometheus data source

---

## Custom Queries

### Request Rate

```promql
rate(golem_agent_requests_total[5m])
```

### Error Rate

```promql
rate(golem_agent_errors_total[5m]) / rate(golem_agent_requests_total[5m])
```

### P95 Latency

```promql
histogram_quantile(0.95, rate(golem_agent_latency_seconds_bucket[5m]))
```

### Cost per Hour

```promql
rate(golem_cost_usd_total[1h])
```

### Active Sessions

```promql
golem_sessions_active
```

---

## Alerting

### Example Alert Rules

```yaml
groups:
  - name: golem
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: rate(golem_agent_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"

      # High latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(golem_agent_latency_seconds_bucket[5m])) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High P95 latency"

      # Budget alert
      - alert: BudgetExhausted
        expr: golem_cost_usd_total > 100
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Budget threshold exceeded"
```

---

## Kubernetes Monitoring

### ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: golem
  namespace: golem
spec:
  selector:
    matchLabels:
      app: golem
  endpoints:
    - port: gateway
      path: /metrics
      interval: 15s
```

### PrometheusRule

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: golem-alerts
  namespace: golem
spec:
  groups:
    - name: golem
      rules:
        # ... alert rules here
```

---

## Metrics Endpoint

| Endpoint | Description |
|----------|-------------|
| `/metrics` | Prometheus metrics |
| `/health` | Health check |

---

## Related

- [DEPLOY.md](DEPLOY.md) — Deployment guide
- [GATEWAY-API.md](GATEWAY-API.md) — HTTP API reference
- [internal/metrics/](internal/metrics/) — Metrics implementation
