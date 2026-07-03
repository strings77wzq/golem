## Why

Phase 1 已修复 revive/gosec 权限问题，但 errcheck 仍被禁用（~100 个 unchecked errors），测试覆盖率未达到 80% 门槛。这些问题违反 CLAUDE.md 的硬约束。

## What Changes

### errcheck 启用（核心变更）

对 `fmt.Fprintln`/`fmt.Fprintf` 的 unchecked errors 进行修复：
- CLI 工具的 stdout 写入：检查错误并记录日志
- 不改变用户体验，只增加错误可见性

### 测试覆盖率提升

为低覆盖率包补充核心路径测试：
- `core/database`：32.7% → 60%+（补充 SQLite 核心路径）
- `core/tools/infra`：55.2% → 70%+（补充 exec 路径）
- `foundation/term`：77.8% → 80%+（补充边界测试）

### G204 子进程安全

对 shellhook 和 exec 的子进程调用添加输入验证：
- 路径清理（防止路径遍历）
- 参数白名单验证

## Capabilities

### New Capabilities

- `errcheck-compliance`：所有 error 被检查，符合 Go 最佳实践
- `test-coverage-gates`：所有核心包达到 80% 覆盖率
- `subprocess-security`：子进程调用有输入验证

### Modified Capabilities

- `linter-config`：errcheck 从 disable 移到 enable

## Impact

- **Code**: `cmd/golem/` (100+ 文件), `core/tools/exec/`, `core/tools/infra/`, `foundation/term/`
- **APIs**: 无公共 API 变更
- **Dependencies**: 无新增
- **Risk**: 低。所有变更都是错误处理或添加测试，不改变运行时行为
