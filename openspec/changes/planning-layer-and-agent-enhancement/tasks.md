# Tasks: Cloud-Native AI Agent with Infrastructure & Data Intelligence

## Phase 1: Database Layer (T1-T5)

### T1: Driver interface + SQLite driver
- Create `core/database/driver.go` with Driver interface, Config, Row, Result types
- Create `core/database/sqlite.go` with SQLite driver (modernc.org/sqlite)
- Implement Connect, Query, Execute, GetSchema, Ping
- Create `core/database/sqlite_test.go`
- Seed data (users, articles, comments)

### T2: Database Registry
- Create `core/database/registry.go` with Registry
- Implement Register, Connect, Get, List, Close
- Create `core/database/registry_test.go`

### T3: MySQL driver
- Create `core/database/mysql.go`
- Implement MySQL driver using go-sql-driver/mysql
- Connection pooling, query timeout
- Create `core/database/mysql_test.go`

### T4: PostgreSQL driver
- Create `core/database/postgres.go`
- Implement PostgreSQL driver using lib/pq
- Create `core/database/postgres_test.go`

### T5: Redis driver
- Create `core/database/redis.go`
- Implement Redis driver using go-redis/redis
- Support GET, SET, DEL, KEYS, HGETALL, LRANGE, INFO
- Create `core/database/redis_test.go`

## Phase 2: Database Tools (T6-T8)

### T6: sql_query + sql_schema tools
- Create `core/tools/database/sql_query.go`
- Create `core/tools/database/sql_schema.go`
- Multi-database support (specify database parameter)
- Create `core/tools/database/*_test.go`

### T7: sql_analyze + CRUD tools
- Create `core/tools/database/sql_analyze.go`
- Create `core/tools/database/sql_insert.go`
- Create `core/tools/database/sql_update.go`
- Create `core/tools/database/sql_delete.go`
- Safety guards (--allow-writes, --confirm-delete)

### T8: Wire database into agent
- Add `--db` flag to `golem agent`
- Register database tools when `--db` is set
- Inject schema into system prompt
- Test: end-to-end with SQLite

## Phase 3: Infrastructure Tools (T9-T11)

### T9: Docker tools
- Create `core/tools/infra/docker.go`
- Implement docker_build, docker_ps, docker_logs, docker_run, docker_stop, docker_exec
- Safety tiers (read-only always, writes with flag)
- Create `core/tools/infra/docker_test.go`

### T10: Kubernetes tools
- Create `core/tools/infra/kubectl.go`
- Implement k8s_get, k8s_apply, k8s_delete, k8s_describe, k8s_logs, k8s_scale
- Safety tiers
- Create `core/tools/infra/kubectl_test.go`

### T11: Helm tools
- Create `core/tools/infra/helm.go`
- Implement helm_install, helm_upgrade, helm_list, helm_rollback
- Safety tiers
- Create `core/tools/infra/helm_test.go`

## Phase 4: Planner (T12-T14)

### T12: Plan data model
- Already implemented (core/planner/plan.go)
- Verify tests pass

### T13: Planner.Decompose + Revise
- Already implemented (core/planner/planner.go)
- Verify tests pass

### T14: Planner integration
- Wire planner into agent loop
- Add complexity detection
- Add --plan CLI flag
- Test with mock provider

## Phase 5: Tool Selector & Reflector (T15-T16)

### T15: Tool Selector
- Create `core/agent/tool_selector.go`
- Keyword-based relevance scoring
- Create `core/agent/tool_selector_test.go`

### T16: Reflector
- Create `core/agent/reflector.go`
- Heuristic evaluation + LLM escalation
- Create `core/agent/reflector_test.go`

## Phase 6: Agent Integration (T17-T19)

### T17: Enhanced agent loop
- Modify `core/agent/loop.go` for planning mode
- Mini ReAct loop per step
- Test with mock provider

### T18: Agent configuration
- Add PlanningConfig to config.go
- Add --plan, --allow-writes, --allow-infra, --confirm-destructive flags
- Test: config loading

### T19: Progress display + session integration
- Emit progress messages during plan execution
- Store ActivePlan in session
- Test: end-to-end

## Phase 7: Polish (T20-T21)

### T20: Documentation
- Update README with all new flags
- Add database + infrastructure sections to docs
- Update architecture diagram

### T21: Integration test
- End-to-end: user request → plan → SQL query → Docker build → K8s deploy → verify
- Test with real SQLite + Docker
