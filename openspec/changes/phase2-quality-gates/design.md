## Context

Phase 1 已修复 revive/gosec 权限问题，但 errcheck 仍被禁用。当前 `golangci.yaml` 禁用了 errcheck，导致所有 unchecked errors 不被检测。

约束（CLAUDE.md）：
- "所有 error 必须显式处理"
- "测试覆盖率 ≥ 80%"
- "声明完成 = 门禁全绿"

## Goals / Non-Goals

**Goals:**
- G1: 启用 errcheck，修复所有 unchecked errors
- G2: 补充低覆盖率包测试到 80%+
- G3: 对 shellhook/exec 添加子进程输入验证
- G4: 保持向后兼容，不改变用户体验

**Non-Goals:**
- 修复所有 gosec G204（子进程）— 需要架构变更
- 达到 100% 覆盖率 — 过度工程
- 改变 CLI 工具的输出格式

## Decisions

### D1 — errcheck 修复策略：检查错误并记录日志

**决策**：对 `fmt.Fprintln`/`fmt.Fprintf` 的 unchecked errors，添加错误检查并记录日志。

**理由**：
1. 符合"所有 error 必须显式处理"原则
2. 错误可见但不改变用户体验
3. 符合 Go 惯例

**实现**：
```go
// Before
fmt.Fprintln(w, "hello")

// After
if _, err := fmt.Fprintln(w, "hello"); err != nil {
    log.Error("write failed", "error", err)
}
```

**替代方案**：
- A: `//nolint:errcheck` — 否决，违反"error 必须处理"原则
- C: 创建 helper 函数 — 否决，过度工程

### D2 — 测试覆盖率策略：核心路径优先

**决策**：为每个低覆盖率包补充核心路径测试。

**理由**：
1. 核心路径测试 ROI 最高
2. 边界测试可后续补充
3. 80% 是合理目标

**实现**：优先测试：
1. `core/database/sqlite.go`：Connect/Query/Execute/GetSchema
2. `core/tools/exec/exec.go`：Execute（sandbox 验证）
3. `foundation/term/detect.go`：IsTerminal/ReadLine

### D3 — G204 策略：混合策略

**决策**：对 shellhook 和 exec 添加输入验证，其他用 #nosec。

**理由**：
1. shellhook/exec 接受用户/LLM 输入，需要验证
2. docker/mcp 是用户自己的责任，可以用 #nosec
3. 平衡安全和实现成本

**实现**：
```go
// shellhook.go - 添加路径清理
func validateCommand(command string) error {
    // 清理路径，防止路径遍历
    clean := filepath.Clean(command)
    if strings.Contains(clean, "..") {
        return fmt.Errorf("command contains path traversal")
    }
    return nil
}
```

## Risks / Trade-offs

- [errcheck 修复] 可能引入新的错误处理逻辑 → 需要充分测试
- [G204 输入验证] 可能过度限制合法命令 → 需要白名单机制
- [测试覆盖] 某些路径难以测试 → 接受 80% 作为目标
