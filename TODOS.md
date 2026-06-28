# TODOS

Captured but not in scope for the current change (`product-positioning-and-oss-excellence`).
Each entry has enough context that someone picking it up in 3 months understands the why.

---

## ~~Local LLM provider (Ollama)~~ — COMPLETED

**Status:** ✅ Implemented and wired. Provider adapter at `core/providers/ollama/`, registered via init(), onboard preset "Ollama (local, no key needed)".

**What was done:**
- `core/providers/ollama/ollama.go` — wraps OpenAI adapter with Ollama defaults (no API key, localhost base)
- `core/providers/ollama/register.go` — init() registration in GlobalRegistry
- `core/providers/ollama/ollama_test.go` — 3 tests (New, WithAPIBase, HealthCheckUnreachable)
- `internal/wiring/providers.go` — blank import triggers init()
- `cmd/golem/onboard.go` — "Ollama (local, no key needed)" preset + connectivity validation
- HealthCheck queries `/api/tags` for model discovery
- ListModels returns installed model names

---

## Full metrics measurement loop

**What:** Implement the measurement infrastructure called out in `docs/PHASE4-METRICS.md`: install success rate, time-to-first-success, quickstart completion rate, docs click-through.

**Why:** The current change captures a baseline (slice 1) and updates README/quickstart/docs, but cannot prove the changes actually made adoption easier. A measurement loop converts "default starting point for Go developers" from a slogan into a verifiable claim.

**Pros:**
- Closes the OSS-excellence loop: ship → measure → iterate.
- Provides evidence for future PRs to upgrade maturity matrix labels.
- Aligns with the design.md success criteria.

**Cons:**
- Requires telemetry or analytics infra (Plausible, GitHub Insights, custom).
- Privacy considerations: must be opt-in or aggregate-only.
- Easy to over-engineer.

**Context:**
- `docs/PHASE4-METRICS.md` already lists the metrics.
- `docs/PHASE4-EXECUTION-ENTRY.md` calls this out as remaining phase4 work.
- Slice 1 baseline doc gives the "before" snapshot to compare against.

**Depends on / blocked by:**
- `product-positioning-and-oss-excellence` slice 1 must ship first to set the baseline.

**Source:** Codex outside-voice review (2026-06-13) and existing phase4 planning.

---

## Release artifact pipeline hardening

**What:** Wire `docs/RELEASE-STRATEGY.md` requirements into actual CI: tarball build, checksums, prebuilt binaries for Linux/Termux/macOS/Windows, release-note auto-generation with verification commands.

**Why:** Current CI builds a Docker image and a Linux amd64 binary artifact, but does not produce the multi-platform release set the strategy doc commits to. OSS excellence requires release polish.

**Pros:**
- Users on macOS/Windows/arm64 get prebuilt binaries instead of building from source.
- Checksums protect against tampering.
- Auto-generated release notes reduce human error.

**Cons:**
- CI workflow becomes more complex (matrix build, signing, GitHub Release API).
- Release process gets tied to CI; manual emergency releases harder.

**Context:**
- `docs/RELEASE-STRATEGY.md` defines the artifact set.
- Existing `.github/workflows/ci.yml` only builds Linux amd64.
- Common pattern: GoReleaser handles this with one config file.

**Depends on / blocked by:**
- None. Independent of the positioning change.

**Source:** Codex outside-voice review (2026-06-13).
