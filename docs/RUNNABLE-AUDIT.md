# Runnable Onboarding Audit (Linux amd64 + Termux)

This audit summarizes the current onboarding friction points for first-time users and the priority fixes for a 5-minute first success path.

## Scope

- Linux amd64
- Android/Termux ARM64
- First run path: install → init → verify success

## Current Blockers

1. PATH guidance is easy to miss after `go install`.
2. No explicit verification ladder for first success.
3. Termux-specific path and environment caveats are not centralized.
4. Install methods are present, but decision guidance is weak.
5. First-success demo flow exists in pieces, not as one short script.

## Priority Fixes

### P0

- Add a dedicated 5-minute quickstart page.
- Add explicit verification commands with expected outcomes.

### P1

- Add platform matrix for Linux/Termux/macOS/Windows support and caveats.
- Add first-success reproducible demo flow.

### P2

- Add unified troubleshooting entry for first-run failures.

## Success Criteria

- New user can complete install + init + first successful response within 5 minutes.
- New user can self-diagnose first-run failures using one troubleshooting path.
- Termux users have clear and tested setup guidance without guesswork.
