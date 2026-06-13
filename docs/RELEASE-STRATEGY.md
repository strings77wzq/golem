# Release Strategy (Artifacts + Cadence)

## Cadence

- Minor release every 2-4 weeks when user-facing value accumulates
- Patch release for regressions/security/critical fixes

## Artifact set

- Source tarball + checksums
- Prebuilt binaries for primary platforms
- Release notes with:
  - highlights
  - breaking changes
  - upgrade steps
  - verification commands

## Release note template

1. What's new
2. Why it matters
3. How to upgrade
4. Verify after upgrade
5. Known issues / rollback hints

## Success criteria

- New users can install from release assets without source build
- Existing users can upgrade with one clear command path
