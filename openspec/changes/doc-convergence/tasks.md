## 1. 文档修正

- [ ] 1.1 `feature/rag/pipeline.go:2` 注释：TF-IDF → BM25（措辞对齐 hybrid.go:11）
- [ ] 1.2 项目 `CLAUDE.md` Providers 段：case 分支描述 → registry 动态注册描述
- [ ] 1.3 验证：`go build ./...` 通过（注释无编译影响）；grep 确认无其他 TF-IDF 残留

## 2. 收尾

- [ ] 2.1 `git diff --name-only` 边界核对（仅 2 文件）
- [ ] 2.2 输出 task-output.md
