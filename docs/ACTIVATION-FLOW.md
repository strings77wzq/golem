# Activation Flow (First-Run to Habit)

This document defines the first-run activation path for Golem users.

## Goal

Help a new user reach a meaningful success moment quickly, then guide them to a repeatable second workflow.

## Activation Stages

1. Install
2. Init provider/model
3. First successful response
4. First practical command
5. Second workflow recommendation

## Entry points

- `README.md` quickstart block
- `docs/QUICKSTART.md`
- `docs/FIRST-SUCCESS-DEMO.md`

## Recommended sequence

### Stage A: Immediate success

```bash
golem init
golem agent -m "hello"
```

### Stage B: Practical value

```bash
golem agent -m "用Go写一个快速排序算法"
```

### Stage C: Next-step activation

```bash
golem agent --rag ./docs -m "总结文档重点"
```

## Drop-off prevention

- If command not found -> PATH fix in troubleshooting
- If provider/auth fails -> status + init rewrite
- If TUI confusion -> explain TTY vs non-TTY behavior

## Success indicators

- User reaches first model response
- User runs second workflow from recommendation
- User visits docs/tutorials next-step links
