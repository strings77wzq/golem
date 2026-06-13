# Spec: Infrastructure Tools

## [S1] Docker Tools

### docker_build

Build a Docker image.

**Input:**
```json
{
  "action": "build",
  "context": ".",
  "tag": "golem:latest",
  "dockerfile": "Dockerfile"
}
```

**Output:**
```
Successfully built golem:latest
Image ID: sha256:abc123...
```

**Safety:** Requires `--allow-infra` flag.

### docker_ps

List containers.

**Input:**
```json
{
  "action": "ps",
  "filter": "status=running",
  "all": true
}
```

**Output:**
```
| CONTAINER ID | IMAGE | STATUS | PORTS | NAMES |
|-------------|-------|--------|-------|-------|
| abc123 | golem:latest | Up 2 hours | 0.0.0.0:18790->18790 | golem-1 |
```

**Safety:** Always allowed (read-only).

### docker_logs

Get container logs.

**Input:**
```json
{
  "action": "logs",
  "container": "golem-1",
  "tail": 100,
  "since": "1h"
}
```

**Output:** Container log output.

**Safety:** Always allowed (read-only).

### docker_run

Run a container.

**Input:**
```json
{
  "action": "run",
  "image": "golem:latest",
  "name": "golem-test",
  "ports": ["18790:18790"],
  "env": {"OPENAI_API_KEY": "sk-..."}
}
```

**Output:**
```
Container started: golem-test (id: abc123)
```

**Safety:** Requires `--allow-infra` flag.

### docker_stop

Stop a container.

**Input:**
```json
{
  "action": "stop",
  "container": "golem-test"
}
```

**Safety:** Requires `--allow-infra` flag.

### docker_exec

Execute command in container.

**Input:**
```json
{
  "action": "exec",
  "container": "golem-1",
  "command": ["ls", "-la", "/app"]
}
```

**Safety:** Requires `--allow-infra` flag.

## [S2] Kubernetes Tools

### k8s_get

Get resources.

**Input:**
```json
{
  "action": "get",
  "resource": "pods",
  "namespace": "default",
  "output": "wide"
}
```

**Output:**
```
| NAME | READY | STATUS | RESTARTS | AGE | NODE |
|------|-------|--------|----------|-----|------|
| golem-abc | 1/1 | Running | 0 | 2h | node1 |
```

**Safety:** Always allowed (read-only).

### k8s_apply

Apply YAML manifest.

**Input:**
```json
{
  "action": "apply",
  "file": "deployment.yaml"
}
```

or inline YAML:
```json
{
  "action": "apply",
  "yaml": "apiVersion: apps/v1\nkind: Deployment\n..."
}
```

**Safety:** Requires `--allow-infra` flag.

### k8s_delete

Delete resources.

**Input:**
```json
{
  "action": "delete",
  "resource": "pod",
  "name": "golem-abc",
  "namespace": "default"
}
```

**Safety:** Requires `--allow-infra` AND `--confirm-destructive` flags.

### k8s_describe

Describe a resource.

**Input:**
```json
{
  "action": "describe",
  "resource": "deployment",
  "name": "golem",
  "namespace": "default"
}
```

**Safety:** Always allowed (read-only).

### k8s_logs

Get pod logs.

**Input:**
```json
{
  "action": "logs",
  "pod": "golem-abc",
  "namespace": "default",
  "tail": 100
}
```

**Safety:** Always allowed (read-only).

### k8s_scale

Scale a deployment.

**Input:**
```json
{
  "action": "scale",
  "deployment": "golem",
  "replicas": 3,
  "namespace": "default"
}
```

**Safety:** Requires `--allow-infra` flag.

## [S3] Helm Tools

### helm_install

Install a Helm chart.

**Input:**
```json
{
  "action": "install",
  "release": "golem",
  "chart": "./helm/golem",
  "namespace": "default",
  "values": {"replicaCount": 2}
}
```

**Safety:** Requires `--allow-infra` flag.

### helm_upgrade

Upgrade a Helm release.

**Input:**
```json
{
  "action": "upgrade",
  "release": "golem",
  "chart": "./helm/golem",
  "set": {"replicaCount": 3}
}
```

**Safety:** Requires `--allow-infra` flag.

### helm_list

List Helm releases.

**Input:**
```json
{"action": "list", "namespace": "default"}
```

**Safety:** Always allowed (read-only).

### helm_rollback

Rollback to previous version.

**Input:**
```json
{
  "action": "rollback",
  "release": "golem",
  "revision": 1
}
```

**Safety:** Requires `--allow-infra` flag.

## [S4] Safety Tiers

| Operation | Always Allowed | --allow-infra | --confirm-destructive |
|-----------|---------------|---------------|----------------------|
| docker ps | ✅ | | |
| docker logs | ✅ | | |
| docker build | | ✅ | |
| docker run | | ✅ | |
| docker stop | | ✅ | |
| docker exec | | ✅ | |
| k8s get | ✅ | | |
| k8s describe | ✅ | | |
| k8s logs | ✅ | | |
| k8s apply | | ✅ | |
| k8s scale | | ✅ | |
| k8s delete | | | ✅ |
| helm list | ✅ | | |
| helm install | | ✅ | |
| helm upgrade | | ✅ | |
| helm rollback | | ✅ | |
