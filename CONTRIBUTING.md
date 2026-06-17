# Contributing to Golem

Thanks for your interest in contributing to Golem!

## Prerequisites

- Go 1.25+
- Git
- Linux amd64 or Android/Termux ARM64 preferred for local verification
- `CGO_ENABLED=0` is mandatory

Recommended reading before first PR:

- Project overview: `README.md`
- Architecture and guardrails: `AGENTS.md`
- Documentation index: `docs/README.md`

## Development Setup

```bash
git clone https://github.com/strings77wzq/golem.git
cd golem
make build
go test ./...
```

## Available Commands

| Command | Description |
|---------|-------------|
| `make build` | Build binary to `build/golem` (pure Go, CGO_ENABLED=0) |
| `make build-all` | Cross-compile for linux/darwin amd64/arm64 |
| `make test` | Run all tests with race detection |
| `make lint` | Run golangci-lint (if installed) |
| `make deps` | Download and verify dependencies |
| `make clean` | Remove build artifacts |
| `make fmt` | Format code with gofmt |
| `make vet` | Run go vet |
| `make check` | Full CI check: deps + vet + test |

## Quick Reference

```bash
# Build
CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o build/golem ./cmd/golem

# Test
go test ./...                                    # all packages
go test -race ./...                              # with race detector
go test -coverprofile=coverage.out ./...         # with coverage

# First-run setup
./build/golem init
./build/golem agent -m "Hello"
```

## Development Workflow

1. Fork repository and create a feature branch.
2. Implement the change with tests and docs updates if needed.
3. Run verification commands.
4. Open a pull request with clear scope and motivation.

```bash
git checkout -b feat/my-change
go test -race ./...
go vet ./...
```

## Verification Checklist

Before PR submission:

- [ ] Build succeeds with pure Go settings
- [ ] Related tests are added/updated
- [ ] `go test -race ./...` passes
- [ ] `go vet ./...` passes
- [ ] Docs are updated for any user-facing changes

## Commit Message Format

We follow Conventional Commits:

```text
<type>(<scope>): <description>
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `build`, `ci`

Examples:

- `feat(agent): add auto-compaction trigger`
- `fix(bm25): handle duplicate document IDs`
- `docs(readme): update architecture diagram`

## Architecture and Code Standards

- Keep layer boundaries strict (cmd/core/foundation/feature/internal)
- No circular imports
- No CGO dependencies
- Return errors instead of panics in library code
- Wrap errors with context (`fmt.Errorf("...: %w", err)`)

## Code Structure Index

- `cmd/golem/`: composition root and CLI wiring
- `core/`: domain logic (agent, providers, tools, session)
- `foundation/`: infrastructure primitives (logger, store, concurrency)
- `feature/`: optional feature modules (mcp, rag, skills, memory, config)
- `internal/`: channels, gateway, metrics, security adapters
- `docs/`: user docs and study guides
- `openspec/`: specification-driven change artifacts

## Reporting Issues and Requests

Use GitHub issue templates:

- Bug report
- Feature request
- Documentation request

Please include Go version, OS/architecture, and reproducible steps for bugs.

## AI Collaboration Principles

- Understand goals first, then implementation details
- Surface better alternatives when they reduce risk or complexity
- Avoid speculative changes without evidence

## License

By contributing, you agree your contributions are licensed under the MIT License.

---

## 中文说明

[English](#contributing-to-golem) | **中文**

感谢你对 Golem 的关注！

### 前置要求

- Go 1.25+
- Git
- Linux amd64 或 Android/Termux ARM64（推荐本地验证）
- `CGO_ENABLED=0` 强制执行

### 开发流程

1. Fork 仓库并创建特性分支
2. 实现变更，添加测试和文档更新
3. 运行验证命令
4. 提交 PR，清晰描述范围和动机

### 验证清单

- [ ] 纯 Go 设置下构建成功
- [ ] 相关测试已添加/更新
- [ ] `go test -race ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] 用户可见的变更已更新文档

### 核心原则

- 保持严格分层（cmd → wiring → core → foundation）
- 禁止循环导入
- 禁止 CGO 依赖
- 库代码返回错误而非 panic
- 错误包装带上下文（`fmt.Errorf("...: %w", err)`）
