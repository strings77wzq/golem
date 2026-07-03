# Brainstorm: Golem 项目演进路线图

> 2026-07-03 产出。基于 axiom-driven-development 流程，深入源码扫描后综合分析。

---

## Phase 1：提取公理（Axioms）

### 约束（硬边界 — 不可违反）

| # | 约束 | 来源 | 影响 |
|---|------|------|------|
| C1 | `CGO_ENABLED=0` 始终成立 | CLAUDE.md Hard Constraints | 所有依赖必须 pure Go |
| C2 | `LLMProvider`/`StreamingProvider` 接口冻结 | CLAUDE.md / openspec/config.yaml | 新 provider 适配器只能实现，不能改签名 |
| C3 | Bubble Tea 隔离在 `internal/channels/tui/` | CLAUDE.md | 其他包不得导入 bubbletea |
| C4 | 工具注册表字母排序（KV-cache 优化） | CLAUDE.md / AGENTS.md | 不可按注册顺序或随机排序 |
| C5 | 工具调用期间禁用流式 | CLAUDE.md: `canStream := ok && streamFinal && len(toolDefs) == 0` | 设计决策，非 bug |
| C6 | 层依赖规则由 Go import 强制 | openspec/config.yaml / core/architecture/ | `core/` 不得导入 `internal/`/`feature/`/`cmd/` |
| C7 | 3 层数据库安全模型（PermissionChecker → QualityGate → RollbackSQL） | CLAUDE.md | 不可绕过 |
| C8 | 测试覆盖率 ≥ 80% | CLAUDE.md / rules/common/quality-gates.md | 低于此值 = 未完成 |

### 不变式（必须始终成立）

| # | 不变式 | 证据 |
|---|--------|------|
| I1 | `go test -race ./...` 零失败 | ✅ 当前通过 |
| I2 | `go vet ./...` 零 issue | ✅ 当前通过 |
| I3 | 无硬编码密钥/令牌 | ✅ 全部走环境变量 (`VENDOR_API_KEY`/`GOLEM_API_KEY`) |
| I4 | Error 显式处理，不静默吞掉 | ✅ 无 TODO/FIXME/HACK 残留 |
| I5 | 线程安全：所有共享状态用 `sync.RWMutex` 或 `atomic` | ✅ Registry/Bus/Session/Store 均有锁保护 |
| I6 | 工具双通道输出（ForLLM + ForUser）保持一致 | ✅ ToolResult 结构已定义 |

### 假设（可被推翻的前提）

| # | 假设 | 当前状态 | 推翻条件 |
|---|------|---------|---------|
| A1 | "功能模块全部 COMPLETE" | ✅ 代码实现完整 | AGENTS.md 仍标记 `memory` 为 "🚧 In development"，实际已 COMPLETE |
| A2 | "82.5% 覆盖率足够" | ⚠️ 15 个包低于 80% | 生产部署要求核心包 ≥ 85% |
| A3 | "Ollama 适配器足够覆盖本地 LLM" | ✅ 已实现 | 如果需要 GGUF 直接推理（llama.cpp），需要新 provider |
| A4 | "HTTP 网关 :18790 足够" | ⚠️ 无 TLS、无自定义端口配置 | 生产部署需要 TLS + 可配置端口 |
| A5 | "单机 SQLite 会话存储足够" | ✅ 有 SQLite + 内存回退 | 多实例部署需要 Redis/PostgreSQL 会话存储 |
| A6 | "MCP STDIO 传输足够" | ✅ 当前实现 | MCP HTTP/SSE 传输在多用户场景下更合适 |
| A7 | "BM25 + 内存向量存储足够 RAG" | ✅ 当前实现 | 大规模文档需要持久化向量存储（已支持 Qdrant） |

---

## Phase 2：当前状态量化评估

### 代码规模与成熟度

| 指标 | 数值 | 评估 |
|------|------|------|
| 总代码行数 | 47,064 | 中大型项目 |
| Go 文件数 | 290（含 130 测试文件） | 测试文件占比 45% — 优秀 |
| 包数量 | ~43 个（5 层） | 架构清晰 |
| 提交总数 | 252 | 4 个月活跃开发 |
| 版本 | v0.9.1（9 个 release） | 接近 1.0 |
| 测试覆盖率 | 82.5%（加权平均） | 超过 80% 门槛 |
| 活跃 openspec 变更 | 9 个 | 大量进行中工作 |

### 覆盖率热力图（低于 80% 的包 — 需要关注）

| 包 | 覆盖率 | 风险等级 | 说明 |
|---|--------|---------|------|
| `core/database` | **33.2%** | 🔴 高 | 多驱动适配器，依赖外部数据库 |
| `cmd/golem` | **39.6%** | 🟡 中 | 组合根，集成密集 |
| `core/tools/infra` | **55.2%** | 🟡 中 | kubectl/docker/helm，依赖外部 CLI |
| `core/tools/database` | **60.8%** | 🟡 中 | SQL 工具，安全门禁已测但集成路径不足 |
| `core/agent` | **62.6%** | 🟡 中 | 核心循环，17 个测试文件但仍低 |
| `internal/channels/tui` | **63.7%** | 🟢 低 | UI 代码，难以单元测试 |
| `core/tools/fileops` | **64.8%** | 🟢 低 | 文件操作，路径遍历保护已测 |
| `internal/gateway` | **66.2%** | 🟢 低 | HTTP 网关，中间件链 |

### 功能模块完成度矩阵

| 模块 | 实现 | 测试 | 接入 | 文档 | 生产就绪 |
|------|------|------|------|------|---------|
| Agent ReAct Loop | ✅ | ⚠️ 62.6% | ✅ | ✅ | ⚠️ |
| OpenAI Provider | ✅ | ✅ 75.3% | ✅ | ✅ | ✅ |
| Anthropic Provider | ✅ | ✅ 88.6% | ✅ | ✅ | ✅ |
| Ollama Provider | ✅ | ✅ 80.0% | ✅ | ✅ | ✅ |
| SQL Tools | ✅ | ⚠️ 60.8% | ✅ | ✅ | ⚠️ |
| Exec Tool | ✅ | ✅ 88.7% | ✅ | ✅ | ✅ |
| MCP Server | ✅ | ✅ 74.1% | ✅ | ✅ | ✅ |
| RAG Pipeline | ✅ | ✅ 88.7% | ✅ | ✅ | ✅ |
| Skills System | ✅ | ✅ 85.4% | ✅ | ✅ | ✅ |
| Memory | ✅ | ✅ 92.6% | ✅ | ⚠️ | ✅ |
| Routing | ✅ | ✅ 86.6% | ✅ `--routing` flag | ⚠️ 文档未标记 | ✅ |
| Health | ✅ | ✅ 91.3% | ✅ `--health` flag | ⚠️ 文档未标记 | ✅ |
| HTTP Gateway | ✅ | ⚠️ 66.2% | ✅ | ✅ | ⚠️ |
| Telegram | ✅ | ✅ 86.7% | ✅ | ⚠️ | ✅ |
| TUI | ✅ | ⚠️ 63.7% | ✅ | ✅ | ✅ |
| Metrics | ✅ | ✅ 96.8% | ✅ `/metrics` 端点已注册 | ⚠️ 文档未标记 | ✅ |

> ⚠️ = 存在差距，需要关注

---

## Phase 3：问题诊断 — 从公理推导差距

### 差距 1：AGENTS.md 与代码状态不一致

**现象**：AGENTS.md 仍将 `memory` 标记为 "🚧 In development"，但代码已 COMPLETE（92.6% 覆盖率）。
**违反公理**：代码验证原则 — 代码是唯一可信来源。
**影响**：误导 AI 协作工具和新贡献者。

### 差距 2：已接入模块文档未标记

**现象**：
- `feature/routing/` — 代码完整（86.6%），**已通过 `--routing` flag 接入**（`cmd/golem/main.go:199-208`），但 AGENTS.md 未标记
- `feature/health/` — 代码完整（91.3%），**已通过 `--health` flag 接入**（`cmd/golem/main.go:213-221`），但 AGENTS.md 未标记
- `foundation/metrics/` — 代码完整（96.8%），**已在 gateway 注册 `/metrics` 端点**（`internal/gateway/routes.go:71`），但文档未说明

**违反公理**：代码验证原则 — 不应假设"文档未标记 = 未接入"。
**影响**：用户不知道这些功能已可用，误以为需要额外开发。

### 差距 3：核心包覆盖率不达标

**现象**：15 个包低于 80%，其中 `core/database`（33.2%）和 `cmd/golem`（39.6%）严重不足。
**违反公理**：C8（覆盖率 ≥ 80%）。
**影响**：生产部署信心不足，回归风险高。

### 差距 4：生产部署基础设施不完整

**现象**：
- 无 TLS 支持（网关仅 HTTP）— 确认 `internal/gateway/` 无 TLS 相关代码
- 无自定义端口配置 — 端口 `:18790` 硬编码
- CI 测试跑 Linux + macOS（`.github/workflows/ci.yml:25-26` matrix），但构建产物仅 ubuntu-latest
- GoReleaser 已配置 6 平台构建（`.goreleaser.yml`: linux/darwin/windows × amd64/arm64），但 CI 未自动触发 release
- 有 `docker/Dockerfile` 和 `docker-compose.yml`，但无发布到 registry 的 CI 步骤
- Helm chart 未集成 ServiceMonitor

**违反公理**：A4（生产部署需要 TLS）和 TODOS.md 中的 "Release artifact pipeline hardening"。
**影响**：GoReleaser 配置已就绪是优势，但 TLS 和自动发布仍需补齐。

### 差距 5：文档与实现的 claim 对齐

**现象**：`product-positioning-and-oss-excellence` 提案已识别此问题 — 文档声称的能力必须与实际可运行、可测试、有文档的行为对齐。
**违反公理**：假设 A1（功能模块全部 COMPLETE）可能在文档层面不成立。
**影响**：用户体验落差。

---

## Phase 4：候选演进方案

### 方案 A：质量巩固优先（推荐）

**核心理念**：先稳固再扩展。v0.9.1 → v1.0 的路径是补齐覆盖率 + 修复文档 + 生产硬化。

**阶段 A1：文档同步（1-2 天）**
- 更新 AGENTS.md：memory 标记为 ✅（已验证 92.6% 覆盖率），routing/health/metrics 标记为已接入（已验证 CLI flag wiring）
- 更新 README.md：对齐 product-positioning 提案的 claim discipline
- 更新 CHANGELOG.md：补全 v0.6.0-v0.9.1 的所有变更

**阶段 A2：覆盖率补缺（3-5 天）**
- `core/database`：33.2% → 80% — 使用 sqlmock 或 SQLite 内存数据库测试
- `cmd/golem`：39.6% → 70% — 测试 CLI 命令解析和配置加载（不测交互式 UI）
- `core/tools/database`：60.8% → 80% — 补充安全门禁集成测试
- `core/agent`：62.6% → 80% — 补充 ReAct 循环的边界用例
- 目标：全项目加权覆盖率 ≥ 85%

**阶段 A3：生产硬化（3-5 天）**
- 完成 `phase3-production-readiness` 提案的所有项
- 网关 TLS 支持 + 端口可配置化
- Security headers middleware
- 审计日志
- Metrics 端点文档化（`/metrics` 已注册，需验证 Prometheus 抓取格式）

**阶段 A4：Release 流水线（2-3 天）**
- GoReleaser 已配置 6 平台（`.goreleaser.yml`） — 需要在 CI 中添加 release 触发步骤
- Docker 镜像发布到 GHCR（已有 Dockerfile，需添加 CI push 步骤）
- Helm chart ServiceMonitor 集成
- 自动化 changelog 生成

**Trade-offs**：
- ✅ 稳健，降低 v1.0 发布风险
- ✅ 修复已知技术债务
- ❌ 不增加新功能，用户感知变化小
- ❌ 需要 10-15 天密集工作

**Socratic 检验**：
- 假设 A2 被推翻（覆盖率不足）→ 方案直接修复此问题 ✅
- 假设 A4 被推翻（无 TLS）→ 方案 A3 修复此问题 ✅
- 为什么此方案而非 B？→ 功能扩展前必须先稳固基础，否则技术债复利增长

---

### 方案 B：功能扩展优先

**核心理念**：用户需要新能力。先加功能，再补质量。

**阶段 B1：MCP HTTP/SSE 传输（3-4 天）**
- 实现 MCP over HTTP（替代纯 stdio）
- 支持多用户并发访问
- 认证集成

**阶段 B2：新 Provider 适配器（2-3 天）**
- Google Gemini（OpenAI 兼容层）
- Mistral（OpenAI 兼容层）
- 本地 GGUF 推理（llama.cpp 集成）

**阶段 B3：高级 RAG（3-5 天）**
- 文档解析器（PDF, DOCX, Markdown）
- 增量索引更新
- 多模态检索（图片+文本）

**阶段 B4：Plugin 系统（3-5 天）**
- 动态加载 Go plugin（需 CGO — 违反 C1）
- 或 WASM plugin runtime
- 或 HTTP webhook tool 扩展

**Trade-offs**：
- ✅ 用户可见新功能
- ✅ 扩大适用场景
- ❌ 违反 C8（覆盖率已不足，扩展会更恶化）
- ❌ B4 可能违反 C1（CGO_ENABLED=0）
- ❌ 技术债继续积累

**Socratic 检验**：
- 假设 A2 被推翻 → 方案 B 不修复此问题 ❌
- 假设 C1 被违反（B4 需要 CGO）→ 方案不可行 ❌
- 为什么否决？→ 在覆盖率不足时扩展功能 = 在沙上建楼

---

### 方案 C：混合策略 — 质量骨架 + 单点功能

**核心理念**：质量是底线，但允许一个高价值功能并行。

**阶段 C1：覆盖率补缺（并行，2-3 天）**
- 只修复 🔴 高风险包：`core/database` 和 `cmd/golem`
- 目标：无包低于 60%

**阶段 C2：文档同步（并行，1 天）**
- 更新 AGENTS.md 状态标记
- 对齐 product-positioning claim

**阶段 C3：单点功能 — MCP HTTP 传输（3-4 天）**
- 最高用户价值的功能扩展
- 不违反任何公理
- 为多用户网关场景铺路

**阶段 C4：Release 流水线（2 天）**
- GoReleaser 基础配置
- 多平台构建

**Trade-offs**：
- ✅ 平衡质量和功能
- ✅ 用户可见改进（MCP HTTP）
- ✅ 风险可控
- ❌ 覆盖率提升不如方案 A 彻底
- ❌ 生产硬化推迟

**Socratic 检验**：
- 假设 A6 被推翻（需要 MCP HTTP）→ 方案直接解决 ✅
- 假设 A2 被推翻 → 方案部分修复（高风险包）⚠️
- 为什么此方案而非 A？→ 如果用户最需要 MCP HTTP，则此方案价值更高

---

## Phase 5：Decision

### 推荐：方案 A（质量巩固优先）

**理由**：

1. **公理对齐**：方案 A 直接修复 C8（覆盖率）、A4（TLS）、A1（文档一致性）三个被违反的公理
2. **技术债复利**：覆盖率不足时扩展功能，每新增一行代码都在增加回归风险
3. **v1.0 门槛**：从 v0.9.1 到 v1.0 应该是质量跃迁，不是功能堆叠
4. **已有 9 个活跃 openspec**：其中 3 个（phase1/2/3 quality gates）直接对应方案 A 的工作
5. **TODOS.md 已捕获**：Release artifact pipeline hardening 已在 backlog，方案 A4 直接执行

### 实施优先级

```
Week 1:
├── Day 1-2: 文档同步（AGENTS.md + README.md + CHANGELOG.md）
├── Day 3-5: core/database 覆盖率 33.2% → 80%
└── Day 5-7: cmd/golem 覆盖率 39.6% → 70%

Week 2:
├── Day 8-9: core/tools/database + core/agent 覆盖率补缺
├── Day 10-11: 生产硬化（TLS + security headers + audit）
└── Day 12-14: GoReleaser + Docker + Helm chart
```

### 里程碑定义

| 里程碑 | 验收标准 |
|--------|---------|
| **M1: 文档同步** | AGENTS.md 所有模块状态与代码一致；README claim 可验证 |
| **M2: 覆盖率达标** | 无包低于 60%；加权平均 ≥ 85%；`go test -race ./...` 通过 |
| **M3: 生产硬化** | TLS 可配置；security headers 生效；审计日志输出；metrics 端点可用 |
| **M4: Release 就绪** | GoReleaser CI 触发自动构建 6 平台；Docker 镜像可拉取；Helm chart 可安装 |

### 否决方案记录

| 方案 | 否决理由 |
|------|---------|
| B（功能扩展优先） | 违反 C8（覆盖率不足时扩展 = 风险累积）；B4 可能违反 C1 |
| C（混合策略） | 覆盖率修复不彻底；生产硬化推迟；与已有 phase1/2/3 openspec 不对齐 |

---

## Phase 6：Socratic 自审清单

- [ ] 假设 A1（文档状态准确）被推翻 → 方案 A1 直接修复 ✅
- [ ] 假设 A2（覆盖率足够）被推翻 → 方案 A2 直接修复 ✅
- [ ] 假设 A4（无 TLS）被推翻 → 方案 A3 直接修复 ✅
- [ ] 边界覆盖：空值 → 测试已覆盖（MockProvider 返回空）✅
- [ ] 边界覆盖：错误路径 → 测试已覆盖（工具错误反馈机制）✅
- [ ] 边界覆盖：并发安全 → `go test -race` 强制 ✅
- [ ] 每个 claim 有证据支撑 → 覆盖率数据来自 `go test -cover` 输出 ✅
- [ ] 为什么此方案而非被否方案 → 已在 Decision 段落说明 ✅
