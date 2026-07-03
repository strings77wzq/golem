## 1. errcheck 启用与修复

- [ ] 1.1 修改 `.golangci.yaml`：从 disable 列表移除 errcheck
- [ ] 1.2 运行 `golangci-lint run --enable errcheck` 获取所有 unchecked errors
- [ ] 1.3 按文件分组修复 errors（优先 cmd/golem/ 入口层）
- [ ] 1.4 修复 `cmd/golem/config.go` 中的 unchecked errors
- [ ] 1.5 修复 `cmd/golem/config_validate.go` 中的 unchecked errors
- [ ] 1.6 修复 `cmd/golem/debug.go` 中的 unchecked errors
- [ ] 1.7 修复 `cmd/golem/demo.go` 中的 unchecked errors
- [ ] 1.8 修复 `cmd/golem/main.go` 中的 unchecked errors
- [ ] 1.9 修复 `cmd/golem/session.go` 中的 unchecked errors
- [ ] 1.10 修复其他 cmd/golem/*.go 中的 unchecked errors
- [ ] 1.11 修复 `core/` 和 `feature/` 中的关键 unchecked errors
- [ ] 1.12 运行 `go build ./...` 确认编译通过
- [ ] 1.13 运行 `go test -race ./...` 确认测试通过

## 2. 测试覆盖率提升

- [ ] 2.1 补充 `core/database/sqlite.go` 核心路径测试（目标 60%+）
- [ ] 2.2 补充 `core/tools/exec/exec.go` 测试（sandbox 验证，目标 70%+）
- [ ] 2.3 补充 `foundation/term/detect.go` 测试（目标 80%+）
- [ ] 2.4 运行 `go test -cover ./...` 确认覆盖率提升

## 3. G204 子进程安全

- [ ] 3.1 分析 `core/agent/shellhook.go` 的输入来源
- [ ] 3.2 为 shellhook 添加命令验证函数
- [ ] 3.3 为 `core/tools/exec/exec.go` 添加路径清理
- [ ] 3.4 添加验证函数的单元测试
- [ ] 3.5 为 docker/mcp 添加 #nosec 注释（已审核）

## 4. 验证与门禁

- [ ] 4.1 `go build ./...` 无错误
- [ ] 4.2 `go vet ./...` 无错误
- [ ] 4.3 `go test -race ./...` 全部通过
- [ ] 4.4 `golangci-lint run` 无新增警告
- [ ] 4.5 所有核心包覆盖率 ≥ 80%
- [ ] 4.6 提交代码，推送，轮询 CI 直到通过
