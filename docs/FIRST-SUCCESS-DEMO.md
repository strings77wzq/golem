# First Success Demo

This is the shortest reproducible flow to prove Golem works.

## Demo Goal

From clean install to a successful agent response in under 5 minutes.

## Step 1: Install

```bash
go install github.com/strings77wzq/golem/cmd/golem@latest
export PATH="$HOME/go/bin:$PATH"
```

## Step 2: Initialize config

```bash
golem init
```

Pick one provider and finish API key setup.

## Step 3: Validation ladder

```bash
golem version
golem status
golem agent -m "hello"
```

### Expected outcome

- `golem version` prints version and build info.
- `golem status` returns config/runtime summary.
- `golem agent -m "hello"` returns a model response without error.

## Step 4: Try one practical command

```bash
golem agent -m "用Go写一个快速排序算法"
```

If this succeeds, your first success path is complete.

## Optional next step

```bash
golem agent
```

Enter interactive mode and continue exploration.
