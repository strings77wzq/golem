# First-Use Workflows

These workflows are designed for rapid adoption and repeat usage.

## Workflow 1: Ask and Get Code

```bash
golem agent -m "用Go写一个快速排序算法，并解释时间复杂度"
```

## Workflow 2: Summarize Local Docs (RAG)

```bash
golem agent --rag ./docs -m "总结这份项目文档，给我一个学习路径"
```

## Workflow 3: Skill-assisted review

```bash
golem agent --skills summarize,code-review -m "请总结并评审这段实现思路"
```

## Workflow 4: Session continuity

```bash
golem agent -C last
```

Use this to continue previous context instead of starting from scratch.

## Recommended order

1. Workflow 1 (basic confidence)
2. Workflow 2 (practical local value)
3. Workflow 3 (advanced utility)
4. Workflow 4 (habit-forming continuity)
