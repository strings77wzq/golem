## Why

项目当前存在 438 个 linter 警告（gosec/revive/prealloc），5 个包覆盖率低于 40%。这违反了 CLAUDE.md 的硬约束：
- "Lint 零新增 error/warning"
- "测试覆盖率 ≥ 80%"
- "声明完成 = 门禁全绿"

这些问题导致：
1. 安全漏洞无法被自动检测（gosec 被部分禁用）
2. 低覆盖率模块的 bug 不会被测试捕获
3. 项目无法声称符合自己定义的质量标准

## What Changes

### 核心变更

- **修复 revive unused-parameter**：将 Cobra 命令函数的未使用参数重命名为 `_`
- **修复 gosec 权限问题**：将 `0755`/`0644` 改为 `0750`/`0600`
- **添加 #nosec 注释**：对已审核的 G304/G404 用法添加注释
- **补充低覆盖率包测试**：为 5 个低覆盖率包添加核心路径测试

### 次要变更

- 启用 errcheck linter（验证无实际问题）
- 同步 golangci.yaml 配置

## Capabilities

### New Capabilities

- `quality-gates-compliance`: 项目符合 CLAUDE.md 定义的质量标准
- `security-coverage`: gosec 在所有非排除路径启用
- `test-coverage`: 所有包覆盖率 ≥ 80%

### Modified Capabilities

- `linter-config`: golangci.yaml 启用 errcheck，修复 exclude 规则

## Impact

- **Code**: `cmd/golem/` (15+ 文件), `core/database/`, `core/tools/infra/`, `foundation/term/`, `core/providers/ollama/`
- **APIs**: 无公共 API 变更
- **Dependencies**: 无新增
- **Risk**: 低。所有变更都是修复 linter 警告或添加测试，不改变运行时行为
