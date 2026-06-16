# Contributing to Golem

感谢你对 Golem 的关注！

## 前置要求

- Go 1.25+
- Git
- Linux amd64 或 Android/Termux ARM64（推荐本地验证）
- `CGO_ENABLED=0` 强制执行

建议在提交第一个 PR 前阅读：

- 项目概览：`README.md`
- 架构与规范：`AGENTS.md`
- 文档索引：`docs/README.md`

## 开发环境

```bash
git clone https://github.com/strings77wzq/golem.git
cd golem
make build
go test ./...
```

## 可用命令

| 命令 | 说明 |
|------|------|
| `make build` | 构建二进制到 `build/golem`（纯 Go，CGO_ENABLED=0） |
| `make build-all` | 交叉编译 linux/darwin amd64/arm64 |
| `make test` | 运行所有测试（含竞态检测） |
| `make lint` | 运行 golangci-lint（如已安装） |
| `make deps` | 下载并验证依赖 |
| `make clean` | 清理构建产物 |
| `make fmt` | gofmt 格式化代码 |
| `make vet` | 运行 go vet |
| `make check` | 完整 CI 检查：deps + vet + test |

## 快速参考

```bash
# 构建
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o build/golem ./cmd/golem

# 测试
go test ./...                                    # 所有包
go test -race ./...                              # 含竞态检测
go test -coverprofile=coverage.out ./...         # 含覆盖率

# 首次运行
./build/golem init
./build/golem agent -m "Hello"
```

## 开发流程

1. Fork 仓库并创建特性分支
2. 实现变更，添加测试和文档更新
3. 运行验证命令
4. 提交 PR，清晰描述范围和动机

```bash
git checkout -b feat/my-change
go test -race ./...
go vet ./...
```

## 验证清单

提交 PR 前：

- [ ] 纯 Go 设置下构建成功
- [ ] 相关测试已添加/更新
- [ ] `go test -race ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] 用户可见的变更已更新文档

## Commit Message 格式

遵循 Conventional Commits：

```text
<type>(<scope>): <description>
```

类型：`feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`

示例：

- `feat(agent): add auto-compaction trigger`
- `fix(bm25): handle duplicate document IDs`
- `docs(readme): update architecture diagram`

## 架构与代码规范

- 保持严格分层（cmd/core/foundation/feature/internal）
- 禁止循环导入
- 禁止 CGO 依赖
- 库代码返回错误而非 panic
- 错误包装带上下文（`fmt.Errorf("...: %w", err)`）

## 代码结构索引

- `cmd/golem/`: 组合根与 CLI 接线
- `core/`: 领域逻辑（agent, providers, tools, session）
- `foundation/`: 基础设施原语（logger, store, concurrency）
- `feature/`: 可选功能模块（mcp, rag, skills, memory, config）
- `internal/`: 通道、网关、指标、安全适配器
- `docs/`: 用户文档与学习指南
- `openspec/`: 规格驱动的变更产物

## 报告问题

使用 GitHub issue 模板：

- Bug 报告
- 功能请求
- 文档请求

请包含 Go 版本、OS/架构、可复现的步骤。

## AI 协作原则

- 先理解目标，再关注实现细节
- 发现更优方案时主动提出
- 避免无证据的投机性变更

## 许可证

贡献即表示你同意你的贡献在 MIT 许可证下授权。
