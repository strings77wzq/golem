# Golem Quick Start (5 Minutes)

This guide gets you from zero to a successful first response with the least possible setup friction.

## Who this guide is for

- New users on Linux amd64
- New users on Android/Termux ARM64

## 1) Install

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
```

If command is not found:

```bash
export PATH="$HOME/go/bin:$PATH"
```

## 2) Initialize provider and model

```bash
golem init
```

Choose one provider preset and fill in the API key.

## 3) Validate your setup (success ladder)

```bash
golem version
golem status
golem agent -m "hello"
```

Expected result:

- `golem version` prints version/build info
- `golem status` shows config + runtime health
- `golem agent -m "hello"` returns a successful model response

## 4) First practical command

```bash
golem agent -m "用Go写一个快速排序算法"
```

If this works, your first success path is complete.

## 5) Next steps

```bash
# Interactive mode (TUI on TTY)
golem agent

# RAG with local docs
golem agent --rag ./docs -m "总结文档重点"

# Skills
golem agent --skills summarize,code-review -m "请总结并评审"
```

## 6) If something fails

- Troubleshooting ladder: `docs/TROUBLESHOOTING.md`
- Hands-on labs: `docs/BEGINNER-LABS.md`
