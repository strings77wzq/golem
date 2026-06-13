## 0. Immediate alignment gates (Slice 1)

- [ ] 0.1 [Trust][Claim Discipline] Remove or soften README/docs first-screen language that overclaims "production-ready," "生产可用," "Cloud-Native," security, metrics, monitoring maturity before evidence exists.
- [ ] 0.2 [Activation][Discoverability] Capture pre-change baseline snapshot in `docs/baselines/2026-06-13-positioning.md`: current README hero text, current quickstart steps, current install commands, current docs/ file count. Future iterations compare against this.
- [ ] 0.3 [Execution Safety] Confirm the risk register and triage rules in `design.md` are honored: cut mock-mode implementation before cutting claim discipline if scope pressure appears.

## 1. Product positioning foundation (Slice 1)

- [ ] 1.1 [Discoverability] Rewrite README top path English-first: hero, value props, install, quickstart, docs entry. Preserve and link clearly to Chinese study content via zh-CN/study links. The boundary "top path" means the contiguous English-first section above any deep links into per-feature guides.
- [ ] 1.2 [Discoverability] Define the canonical tagline and message hierarchy for GitHub description, README hero, docs index, release notes. Tagline: "Go-native, local-first AI agent runtime with a path from terminal prototype to deployable gateway."
- [ ] 1.3 [Trust] Document project category boundaries in README and docs: not hosted SaaS, not no-code builder, not Python framework clone, not chatbot-only demo.

## 2. Developer activation journey (Slice 1)

- [ ] 2.1 [Activation] Map the primary journey explicitly in the rewritten README and docs/QUICKSTART.md: run locally → extend with tools/MCP/RAG → expose as gateway → deploy anywhere.
- [ ] 2.2 [Activation] Audit README, quickstart, BEGINNER-LABS, FIRST-SUCCESS-DEMO, ACTIVATION-FLOW for ordering conflicts; reconcile by linking to one canonical quickstart.
- [ ] 2.3 [Activation][Test] **Mock-first zero-key path — TDD.** Extend `core/providers/mock.go` so `MockProvider.Chat` returns `[mock echo] <last user content>` when the response queue is empty. Preserve queue-mode contract: when responses are queued, return queued responses unchanged. Add tests T-test-1 through T-test-7 first; implement to pass.
  - T-test-1 (P0): echo when queue empty, has user message
  - T-test-2 (P0): echo when messages slice empty → `[mock echo] (empty input)`
  - T-test-3 (P0): echo when last user message Content is empty → `[mock echo] (empty input)`
  - T-test-4 (P0): echo when no user-role messages → `[mock echo] (no user input)`
  - T-test-5 (P0 CRITICAL REGRESSION): queue mode wins over echo when responses queued
  - T-test-6 (P0): echo content streams via `ChatStream` (token-by-token)
  - T-test-7 (P0 E2E): `golem agent -M mock/echo -m "hello"` returns prefixed echo, exit 0
- [ ] 2.4 [Activation][Test] Update `core/providers/mock_test.go::TestMockProviderExhaustedQueue` to assert the new echo contract instead of the old "no more responses queued" error.
- [ ] 2.5 [Activation] Update `docs/QUICKSTART.md` to lead with the zero-key mock path: `golem agent -M mock/echo -m "hello"`, then escalate to API-key path. Document mock as runtime validation, NOT real LLM behavior.
- [ ] 2.6 [Proof] Add a README proof slot tied to the first-success path showing a real terminal transcript or "verify it yourself" command block. The transcript MUST contain the `[mock echo]` prefix so it is self-evidently a runtime check.
- [ ] 2.7 [Maintainability] Fix `docs/TESTING.md:165-173` mock provider example to match current `NewMockProvider(name string)` signature.

## 3. Documentation information architecture (Slice 1 + Slice 2)

- [ ] 3.1 [Discoverability] (Slice 1) Reorganize `docs/README.md` index around journey stages: Start → Extend → Deploy → Learn → Contribute, instead of the current 7-section module inventory.
- [ ] 3.2 [Maintainability] (Slice 1) **Doc consolidation pass.** Audit `docs/SITE-IA.md`, `docs/SITE-PAGES-PLAN.md`, `docs/SITE-MODULE-MAPPING.md`, `docs/SITE-I18N-NAV.md`, `docs/SITE-ASSETS-CHECKLIST.md`, `docs/PROOF-ASSETS-PLAN.md`, `docs/GROWTH-ASSETS.md`, `docs/ACTIVATION-FLOW.md`, `docs/FIRST-USE-WORKFLOWS.md`, `docs/RETENTION-LOOPS.md`, `docs/PHASE4-*.md`. For each: pick a canonical home (or merge), mark superseded files with a header pointer, or move to `docs/archive/`. Goal: every product-facing concept has exactly one source-of-truth file.
- [ ] 3.3 [Discoverability] (Slice 2) Define canonical roles for README, docs index, tutorials, architecture guides, examples, release notes in a contributor-facing rule (likely in CONTRIBUTING.md).
- [ ] 3.4 [Discoverability] (Slice 1) Define English-first global surfaces and supported Chinese paths. Existing `docs/study/` becomes the authoritative Chinese learning track, linked from README and docs index but not duplicated in English.

## 4. OSS trust and governance (Slice 2 + minimum CI in Slice 1)

- [ ] 4.1 [Trust] (Slice 1) **Add `golangci-lint` step to `.github/workflows/ci.yml`** using the existing `.golangci.yaml` config. Required to pass on PRs. Fix or document any baseline lint warnings discovered on first run.
- [ ] 4.2 [Trust] (Slice 2) Promote `docs/SECURITY.md` to root-level `SECURITY.md` (or symlink/short-pointer), so GitHub's community profile auto-detects it.
- [ ] 4.3 [Trust] (Slice 2) Add root `SUPPORT.md` that points to existing `docs/COMMUNITY-CHANNELS.md` and the issue templates. Per codex review, the gap is wiring routes, not creating new files.
- [ ] 4.4 [Trust] (Slice 2) Add root `GOVERNANCE.md` explaining decision-making, maintainer roster, classification dispute resolution.
- [ ] 4.5 [Trust] (Slice 2) Update `CONTRIBUTING.md` with the claim-review checklist for README/docs/release/example changes.
- [ ] 4.6 [Trust] (Slice 2) Update release maturity guidance in `docs/RELEASE-STRATEGY.md` and link from `CHANGELOG.md` so users see cadence, compatibility expectations, breaking-change handling.

## 5. Capability-claim alignment (Slice 2)

- [ ] 5.1 [Claim Discipline] Create `docs/PRODUCT-POSITIONING.md` distilling north star, primary/secondary users, non-goals, message hierarchy, tone, claim rules.
- [ ] 5.2 [Claim Discipline] Create `docs/CAPABILITY-MATURITY.md` with the maturity matrix: CLI/TUI, providers, tools, MCP, RAG, memory, skills, gateway, Telegram, Docker, Kubernetes, Helm, health, metrics, security, routing, Termux. Levels: Stable, Beta, In Progress, Planned, Production Path. Each row includes evidence (file path, test name, command, or explicit limitation).
- [ ] 5.3 [Claim Discipline] Define classification dispute resolution: prefer lower maturity label when evidence is ambiguous; maintainer decision is final fallback.
- [ ] 5.4 [Claim Discipline] Sweep public-facing language across README, docs, release notes for any remaining overclaim ("production-ready," "cloud-native," "enterprise-grade") and reclassify per the matrix.
- [ ] 5.5 [Claim Discipline] Add a claim-review checklist to PR template / CONTRIBUTING for README, docs, website, release-note, example, public-messaging changes.

## 6. Agent presentation alignment (Slice 2)

- [ ] 6.1 [Discoverability] Audit agent-related README, docs, tutorials for chatbot-only or feature-list-first framing and reframe around the local-first runtime journey.
- [ ] 6.2 [Claim Discipline] Label every agent-related public claim per the maturity matrix.

## 7. Verification and review (per slice)

- [ ] 7.1 [Verification] After slice 1 implementation: run `openspec validate product-positioning-and-oss-excellence`, `go vet ./...`, `go test -race ./...`, `golangci-lint run ./...`, manual `golem agent -M mock/echo -m "hello"` smoke test.
- [ ] 7.2 [Verification] After slice 1: run `/qa` against the test plan artifact at `~/.gstack/projects/strings77wzq-golem/strin-main-eng-review-test-plan-20260613-174026.md`.
- [ ] 7.3 [Review] After slice 1 implementation: run `/review` for diff-level adversarial review; address any P0/P1 findings before `/ship`.
- [ ] 7.4 [Verification] After slice 1: `/ship` produces v0.6.x release with English-first README and zero-key mock path. CI green including new golangci-lint step.
- [ ] 7.5 [Verification] After slice 2: repeat 7.1-7.4 for slice 2 producing v0.7.x release with positioning charter, maturity matrix, trust files.
- [ ] 7.6 [Verification] Final summary: map completed work back to each spec requirement, maturity dimension, and capability-claim classification.

## Slice boundary summary

```
Slice 1 (target: v0.6.x release, ships first)
  Group 0 (alignment gates): all
  Group 1 (positioning foundation): all
  Group 2 (activation journey): all
  Group 3 (docs IA): 3.1, 3.2, 3.4
  Group 4 (OSS trust): 4.1 (golangci-lint CI only)
  Group 7 (verification): 7.1, 7.2, 7.3, 7.4

Slice 2 (target: v0.7.x release, ships after slice 1 lands)
  Group 3 (docs IA): 3.3
  Group 4 (OSS trust): 4.2, 4.3, 4.4, 4.5, 4.6
  Group 5 (capability-claim alignment): all
  Group 6 (agent presentation): all
  Group 7 (verification): 7.5, 7.6
```
