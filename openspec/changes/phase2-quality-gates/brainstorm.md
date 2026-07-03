# Brainstorm: Phase 2 Quality Gates

> 2026-06-30 产出。基于 axiom-driven-development 流程。

## Axioms（公理 — 第一性原理基础）

### 约束（硬边界）
1. **errcheck 启用后，所有 `fmt.Fprintln`/`fmt.Fprintf` 必须检查返回值** — errcheck 检测规则
2. **不能改变 CLI 工具的用户行为** — 输出到 stdout 的错误处理不能改变用户体验
3. **Go 惯例：stdout 写入失败通常不处理** — 但 CLI 工具应该记录日志
4. **gosec G204 子进程调用需要安全评估** — 不能盲目添加 #nosec
5. **测试覆盖率要达到 80%** — 但某些包依赖外部服务，难以测试

### 不变式（必须始终成立）
1. **所有修改必须通过 `go test -race ./...`**
2. **所有修改必须通过 `golangci-lint run`**
3. **不能引入新的安全漏洞**
4. **不能破坏现有功能**

### 假设（可被推翻的前提）
1. **假设 A**：CLI 工具的 stdout 写入失败可以忽略 → 推翻：网络错误或磁盘满时会丢失输出
2. **假设 B**：G204 子进程调用都是安全的 → 推翻：如果输入来自 LLM，可能被注入恶意命令
3. **假设 C**：80% 覆盖率是合理目标 → 推翻：某些包（如 Postgres/MySQL）需要真实数据库

## Options（候选方案）

### 任务 1：errcheck 启用

#### 方案 A：忽略 stdout 写入错误
- 描述：对 `fmt.Fprintln`/`fmt.Fprintf` 添加 `//nolint:errcheck` 注释
- Trade-offs：简单，但忽略了真实错误
- Socratic 检验：❌ 否决 — 违反"所有 error 必须显式处理"原则

#### 方案 B：检查错误但只记录日志（推荐）
- 描述：`_, _ = fmt.Fprintln(...)` 改为 `if _, err := fmt.Fprintln(...); err != nil { log.Error(...) }`
- Trade-offs：错误可见，但增加了代码量
- Socratic 检验：✅ 通过 — 符合"错误必须显式处理"原则

#### 方案 C：创建 helper 函数
- 描述：创建 `mustFprintln(w io.Writer, args ...any)` 函数，内部处理错误
- Trade-offs：代码最简洁，但隐藏了错误处理
- Socratic 检验：✅ 通过 — 如果 helper 记录日志并 panic on critical errors

### 任务 2：gosec G204 子进程安全

#### 方案 A：全部 #nosec
- 描述：对所有 G204 添加 `// #nosec G204`
- Trade-offs：简单，但忽略了安全风险
- Socratic 检验：❌ 否决 — 违反"安全不能被忽略"

#### 方案 B：输入验证
- 描述：在子进程调用前验证输入（路径清理、参数白名单）
- Trade-offs：安全，但实现复杂
- Socratic 检验：✅ 通过 — 符合"所有输入都不可信"原则

#### 方案 C：混合策略（推荐）
- 描述：对已审核的调用添加 #nosec，对新调用添加输入验证
- Trade-offs：平衡安全和实现成本
- Socratic 检验：✅ 通过 — 符合"最小权限"原则

### 任务 3：测试覆盖率

#### 方案 A：只测试核心路径
- 描述：为每个包添加 3-5 个核心路径测试
- Trade-offs：快速，但覆盖率提升有限
- Socratic 检验：✅ 通过 — 符合"80/20 法则"

#### 方案 B：全面测试
- 描述：为每个函数添加测试，包括边界情况
- Trade-offs：覆盖率高，但工作量大
- Socratic 检验：❌ 否决 — 某些包（如 Postgres/MySQL）需要真实数据库

#### 方案 C：分层测试（推荐）
- 描述：核心路径 100% 测试，边界情况按风险优先级测试
- Trade-offs：平衡覆盖率和成本
- Socratic 检验：✅ 通过 — 符合"风险驱动测试"原则

## Decision

- **选定**：方案 B（errcheck）、方案 C（G204）、方案 C（测试覆盖率）
- **理由**：
  1. 方案 B（errcheck）：最符合 Go 惯例，错误可见
  2. 方案 C（G204）：平衡安全和实现成本
  3. 方案 C（测试）：核心路径全覆盖，边界按风险
- **待确认项**：
  1. errcheck 修改范围：是否包含所有 CLI 工具的 stdout 写入？
  2. G204 审核标准：哪些调用需要输入验证，哪些可以 #nosec？
  3. 测试覆盖率目标：哪些包必须达到 80%，哪些可以豁免？
