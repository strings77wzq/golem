# Proposal: Planning Layer & Agent Enhancement

## Why

Golem is currently a chat-based ReAct agent — it takes user input, runs a blind think→act→observe loop, and returns text. This is functionally equivalent to "a Go program that calls the OpenAI API." There is no meaningful difference between Golem and a 50-line script that sends messages to an LLM and parses tool calls.

To become a serious cloud-native AI agent, Golem needs two capabilities that separate a **chatbot** from an **agent**:

1. **Planning** — the ability to decompose a complex task into a structured plan before executing
2. **Reflection** — the ability to evaluate its own output and decide whether to continue or stop

Without planning, the agent is reactive — it does whatever the LLM suggests on each turn, with no goal awareness. Without reflection, the agent cannot recover from mistakes or know when it has succeeded.

The user's vision: "Golem is a cloud-native AI agent, not a chatbot clone of Claude Code."

## What Changes

### Planning Layer (`core/planner/`)

A new `planner` package that sits between user input and the ReAct loop:

1. **Task Decomposition** — Given a user request, generate a structured plan (list of steps with expected outcomes)
2. **Plan Execution** — Execute steps sequentially, checking each step's result against expectations
3. **Plan Revision** — If a step fails or produces unexpected results, revise the plan before continuing
4. **Plan Serialization** — Plans are serializable to JSON for debugging and session persistence

### Agent Enhancement (`core/agent/`)

Enhancements to the existing agent loop:

1. **Dynamic Tool Selection** — Instead of sending all 20 tools to the LLM (wasting tokens), select the 5-8 most relevant tools based on the current task context
2. **Structured Output Parsing** — Robust parsing of LLM outputs with fallback strategies for malformed JSON
3. **Reflection Loop** — After each tool execution, evaluate: "Did this step achieve its goal? Should I continue, revise, or stop?"
4. **Goal Tracking** — Maintain a clear understanding of what the user wants and measure progress toward it

### Cloud-Native Integration (`core/agent/` + `feature/`)

Capabilities that make Golem uniquely suited for cloud-native environments:

1. **Infrastructure Awareness** — Built-in understanding of K8s, Docker, CI/CD concepts
2. **Multi-Step Workflows** — Execute complex operations like "deploy this service" as a sequence of orchestrated steps
3. **Observability Integration** — Emit structured logs and metrics at each planning/execution step

## Non-Goals

- Do NOT implement multi-agent orchestration (future work)
- Do NOT change the LLMProvider interface
- Do NOT add new external dependencies
- Do NOT break existing CLI/Gateway/SDK functionality

## Impact

- `core/planner/` — new package
- `core/agent/loop.go` — integrate planner into agent loop
- `core/agent/agent.go` — add planner field and configuration
- `core/context/` — planner uses context manager for token budget
- Tests: new `core/planner/*_test.go` + updated agent tests
