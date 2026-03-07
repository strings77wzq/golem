# Deployment Guide

Complete deployment guide for Golem: Docker, Docker Compose, and Kubernetes.

---

## Prerequisites

- Go 1.25+ (for local build)
- Docker 24+
- Docker Compose v2+
- Kubernetes 1.28+ (for K8s deployment)
- Helm 3.14+ (optional, for Helm deployment)

---

## Quick Start

### 1. Build Binary

```bash
# Clone and build
git clone https://github.com/strings77wzq/golem.git
cd golem

# Build for current platform
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o build/golem ./cmd/golem

# Or use Makefile
make build
```

### 2. Run Locally

```bash
# Agent mode (interactive)
./build/golem agent

# One-shot query
./build/golem agent -m "Hello"

# HTTP Gateway
./build/golem gateway
```

---

## Docker Deployment

### Build Image

```bash
docker build -t golem:latest -f docker/Dockerfile .
```

### Run with Docker Compose

```bash
# Start gateway service
docker compose -f docker/docker-compose.yml up gateway -d

# View logs
docker compose -f docker/docker-compose.yml logs -f gateway

# Stop
docker compose -f docker/docker-compose.yml down
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOLEM_PROVIDER` | LLM provider | `openai` |
| `GOLEM_MODEL` | Model name | `gpt-4o` |
| `OPENAI_API_KEY` | OpenAI API key | - |
| `ANTHROPIC_API_KEY` | Anthropic API key | - |
| `DEEPSEEK_API_KEY` | DeepSeek API key | - |

### Configuration

Mount config directory:

```yaml
volumes:
  - ./config:/app/config:ro
```

Create `config/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt4",
      "max_tokens": 8192,
      "system_prompt": "You are Golem."
    }
  },
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-4o",
      "api_key": "${OPENAI_API_KEY}"
    }
  ]
}
```

---

## Kubernetes Deployment

### 1. Create Namespace

```bash
kubectl apply -f k8s/namespace.yaml
```

### 2. Create Secrets

```bash
# Edit secret.yaml with your API keys
kubectl apply -f k8s/secret.yaml
```

### 3. Create ConfigMap

```bash
kubectl apply -f k8s/configmap.yaml
```

### 4. Deploy

```bash
kubectl apply -f k8s/deployment.yaml
```

### 5. Expose Service

```bash
# ClusterIP (default)
kubectl apply -f k8s/service.yaml

# Or LoadBalancer (cloud)
# Edit service.yaml and change type to LoadBalancer

# Or Ingress
kubectl apply -f k8s/ingress.yaml
```

### 6. Verify

```bash
# Check pods
kubectl get pods -n golem

# Check logs
kubectl logs -n golem -l app=golem

# Check service
kubectl get svc -n golem
```

### Resource Limits

| Resource | Request | Limit |
|----------|---------|-------|
| CPU | 100m | 500m |
| Memory | 128Mi | 512Mi |

---

## Helm Deployment (Optional)

### Install Helm

```bash
# Create Helm chart
mkdir -p k8s/helm/golem
cat > k8s/helm/golem/Chart.yaml << 'EOF'
apiVersion: v2
name: golem
description: A progressive Go AI assistant
version: 0.1.0
EOF

# Create values.yaml
cat > k8s/helm/golem/values.yaml << 'EOF'
replicaCount: 2

image:
  repository: golem
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 18790

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

config:
  model_name: gpt4
  max_tokens: 8192
EOF

# Install
helm install golem k8s/helm/golem --namespace golem --create-namespace
```

### Upgrade

```bash
helm upgrade golem k8s/helm/golem
```

### Uninstall

```bash
helm uninstall golem -n golem
```

---

## Monitoring

See [MONITORING.md](MONITORING.md) for detailed monitoring setup.

Quick start:

```bash
# Start monitoring stack
docker compose -f docker/monitoring/docker-compose.monitoring.yml up -d

# Access:
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
```

---

## Health Checks

### Liveness Probe

- Path: `/health`
- Port: 18790
- Initial delay: 5s
- Period: 30s

### Readiness Probe

- Path: `/health`
- Port: 18790
- Initial delay: 3s
- Period: 10s

---

## Troubleshooting

### Pod not starting

```bash
# Check events
kubectl describe pod -n golem <pod-name>

# Check logs
kubectl logs -n golem <pod-name>
```

### Config not loading

```bash
# Verify ConfigMap
kubectl get configmap golem-config -n golem -o yaml
```

### API key missing

```bash
# Verify Secret
kubectl get secret golem-secrets -n golem -o yaml
```

---

## Security

See [SECURITY.md](SECURITY.md) for security best practices.

Key points:
- Store API keys in Kubernetes Secrets
- Use Ingress with TLS
- Enable network policies
- Run as non-root user (default in Dockerfile)

---

## Related

- [CONFIG-REFERENCE.md](CONFIG-REFERENCE.md) — Configuration reference
- [GATEWAY-API.md](GATEWAY-API.md) — HTTP API reference
- [TESTING.md](TESTING.md) — Testing guide
- [MONITORING.md](MONITORING.md) — Monitoring setup
