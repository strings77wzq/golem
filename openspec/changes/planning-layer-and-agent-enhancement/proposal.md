# Proposal: Cloud-Native AI Agent with Infrastructure & Data Intelligence

## Why

Golem is currently a chat-based ReAct agent with no infrastructure awareness and no database capabilities. To become a **cloud-native AI agent**, it needs to:

1. **Operate infrastructure** — Docker containers, Kubernetes deployments
2. **Manage data stores** — SQLite, MySQL, PostgreSQL, Redis, vector databases
3. **Plan complex tasks** — decompose multi-step operations before executing
4. **Reflect on results** — evaluate output and decide whether to continue

The user is a database engineer with a team using K8s and Docker. Golem should be the AI agent that understands their entire stack.

## What Changes

### 1. Database Intelligence (`core/database/`)

Multi-database support with schema awareness:

- **SQLite** — embedded, pure Go (modernc.org/sqlite)
- **MySQL** — remote database operations
- **PostgreSQL** — remote database operations
- **Redis** — cache and key-value operations
- **Vector DB** — semantic search (pgvector, Qdrant, Milvus)

Each database type gets:
- Schema introspection (understand table structure)
- Query execution (run SQL/commands)
- Data analysis (statistics, distribution)
- CRUD operations (with safety guards)

### 2. Infrastructure Tools (`core/tools/infra/`)

Docker and Kubernetes operations:

- **Docker** — build, run, stop, list, logs, exec
- **Kubernetes** — get, apply, delete, describe, logs, scale
- **Helm** — install, upgrade, list, rollback

### 3. Planning Layer (`core/planner/`)

Task decomposition and structured execution:
- Decompose complex tasks into ordered steps
- Execute steps with tool selection per step
- Revise plan when steps fail

### 4. Agent Enhancement (`core/agent/`)

Intelligence improvements:
- Dynamic tool selection per step
- Reflection loop after each step
- Context management with token budgets

## Database Schema

### SQLite (Users, Articles, Comments)

As defined in database-schema/spec.md.

### MySQL (Remote databases)

Connect to external MySQL servers:
```json
{
  "type": "mysql",
  "host": "localhost",
  "port": 3306,
  "user": "root",
  "password": "***",
  "database": "myapp"
}
```

### PostgreSQL (Remote databases)

Connect to external PostgreSQL servers:
```json
{
  "type": "postgres",
  "host": "localhost",
  "port": 5432,
  "user": "postgres",
  "password": "***",
  "database": "myapp"
}
```

### Redis (Cache)

Connect to Redis:
```json
{
  "type": "redis",
  "host": "localhost",
  "port": 6379,
  "password": "***"
}
```

### Vector DB (Semantic search)

Connect to vector databases:
```json
{
  "type": "qdrant",
  "host": "localhost",
  "port": 6333,
  "collection": "documents"
}
```

## Infrastructure

### Docker Operations

- `docker_build` — build image from Dockerfile
- `docker_run` — run container
- `docker_stop` — stop container
- `docker_ps` — list containers
- `docker_logs` — get container logs
- `docker_exec` — execute command in container

### Kubernetes Operations

- `k8s_get` — get resources (pods, services, deployments)
- `k8s_apply` — apply YAML manifest
- `k8s_delete` — delete resources
- `k8s_describe` — describe resource details
- `k8s_logs` — get pod logs
- `k8s_scale` — scale deployment

### Helm Operations

- `helm_install` — install chart
- `helm_upgrade` — upgrade release
- `helm_list` — list releases
- `helm_rollback` — rollback to previous version

## Non-Goals

- Do NOT implement multi-agent orchestration (future work)
- Do NOT change the LLMProvider interface
- Do NOT break existing CLI/Gateway/SDK functionality
- Do NOT implement cloud provider APIs (AWS/GCP/Azure) — focus on self-hosted

## Impact

- `core/database/` — new package (multi-db support)
- `core/tools/database/` — database tools (7 tools)
- `core/tools/infra/` — infrastructure tools (13 tools)
- `core/planner/` — planning layer
- `core/agent/` — enhanced loop
- `core/context/` — schema injection
- `cmd/golem/` — new flags (--db, --plan, --allow-writes)
- Tests: comprehensive test suite for all new packages
