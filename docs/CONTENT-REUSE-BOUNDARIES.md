# Content Reuse Boundaries

Defines where to reuse content across README, docs, tutorials, and site pages.

## Reuse rules

- Command snippets are canonical in `docs/QUICKSTART.md`
- README should keep concise versions and link to canonical docs
- Tutorial steps should reference quickstart commands, not fork syntax unless necessary

## Avoid

- Divergent command examples across files
- Duplicated troubleshooting ladders with inconsistent fixes
- Rewriting architecture principles in multiple places without links

## Single source mapping

- First-run commands: `docs/QUICKSTART.md`
- First success flow: `docs/FIRST-SUCCESS-DEMO.md`
- Troubleshooting: `docs/TROUBLESHOOTING.md`
- Architecture learning: `docs/study/README.md`
