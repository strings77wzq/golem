## 1. 修复 revive unused-parameter

- [ ] 1.1 修复 `cmd/golem/config.go` 中的 unused-parameter（3 处）
- [ ] 1.2 修复 `cmd/golem/config_validate.go` 中的 unused-parameter（1 处）
- [ ] 1.3 修复 `cmd/golem/debug.go` 中的 unused-parameter（2 处）
- [ ] 1.4 修复 `cmd/golem/demo.go` 中的 unused-parameter（1 处）
- [ ] 1.5 修复 `cmd/golem/main.go` 中的 unused-parameter（3 处）
- [ ] 1.6 修复 `cmd/golem/mcp_server.go` 中的 unused-parameter（1 处）
- [ ] 1.7 运行 `go build ./...` 确认编译通过

## 2. 修复 gosec 权限问题

- [ ] 2.1 修复 `cmd/golem/config.go` 中的 `0755` → `0750`（1 处）
- [ ] 2.2 修复 `cmd/golem/config.go` 中的 `0644` → `0600`（1 处）
- [ ] 2.3 修复 `cmd/golem/demo.go` 中的 `0755` → `0750`（1 处）
- [ ] 2.4 修复 `cmd/golem/memory_adapter.go` 中的 `0755` → `0750`（1 处）
- [ ] 2.5 修复 `cmd/golem/session.go` 中的 `0644` → `0600`（1 处）
- [ ] 2.6 运行 `go build ./...` 确认编译通过

## 3. 添加 #nosec 注释

- [ ] 3.1 为 `cmd/golem/config.go` 中的 G304 添加 `#nosec G304`（4 处）
- [ ] 3.2 为 `cmd/golem/config_validate.go` 中的 G304 添加 `#nosec G304`（1 处）
- [ ] 3.3 为 `cmd/golem/debug.go` 中的 G304 添加 `#nosec G304`（1 处）
- [ ] 3.4 为 `cmd/golem/rag_adapter.go` 中的 G304 添加 `#nosec G304`（1 处）
- [ ] 3.5 为 `cmd/golem/status.go` 中的 G304 添加 `#nosec G304`（2 处）
- [ ] 3.6 为 `core/config/loader.go` 中的 G304 添加 `#nosec G304`（1 处）
- [ ] 3.7 为 `cmd/golem/demo.go` 中的 G404 添加 `#nosec G404`（1 处）
- [ ] 3.8 运行 `go build ./...` 确认编译通过

## 4. 补充低覆盖率包测试

- [ ] 4.1 补充 `core/database/sqlite.go` 核心路径测试（Connect, Query, Execute, GetSchema）
- [ ] 4.2 补充 `core/tools/infra/kubectl.go` 基础测试（New, Name, Description, Parameters）
- [ ] 4.3 补充 `core/tools/infra/docker.go` 基础测试（New, Name, Description, Parameters）
- [ ] 4.4 补充 `core/tools/infra/helm.go` 基础测试（New, Name, Description, Parameters）
- [ ] 4.5 补充 `foundation/term/detect.go` 测试（IsTerminal, ReadLine）
- [ ] 4.6 补充 `core/providers/ollama/ollama.go` 测试（HealthCheck, ListModels）
- [ ] 4.7 运行 `go test -race ./...` 确认所有测试通过

## 5. 启用 errcheck linter

- [ ] 5.1 修改 `.golangci.yaml`：从 disable 列表移除 errcheck
- [ ] 5.2 运行 `golangci-lint run --enable errcheck` 确认无问题
- [ ] 5.3 运行 `go build ./...` 确认编译通过

## 6. 验证与门禁

- [ ] 6.1 `go build ./...` 无错误
- [ ] 6.2 `go vet ./...` 无错误
- [ ] 6.3 `go test -race ./...` 全部通过
- [ ] 6.4 `golangci-lint run` 无新增警告
- [ ] 6.5 所有包覆盖率 ≥ 80%
- [ ] 6.6 提交代码，推送，轮询 CI 直到通过
