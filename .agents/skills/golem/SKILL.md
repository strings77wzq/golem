```markdown
# golem Development Patterns

> Auto-generated skill from repository analysis

## Overview

This skill teaches the core development patterns and workflows of the `golem` repository, a Go codebase focused on modular, testable, and spec-driven development. It covers coding conventions, commit practices, and the main workflows for proposal lifecycle management, feature/bugfix development with tests, and end-to-end (E2E) test scaffolding. By following these guidelines, contributors can ensure consistency, traceability, and high code quality across the project.

## Coding Conventions

- **File Naming:**  
  Use `camelCase` for Go source files.  
  _Example:_  
  ```
  myModule.go
  myModule_test.go
  ```

- **Import Style:**  
  Use relative imports within the module.  
  _Example:_  
  ```go
  import (
      "core/utils"
      "internal/helpers"
  )
  ```

- **Export Style:**  
  Use named exports for functions, types, and variables.  
  _Example:_  
  ```go
  // Exported function
  func DoWork() error {
      // ...
  }
  ```

- **Commit Messages:**  
  Follow [Conventional Commits](https://www.conventionalcommits.org/) with prefixes: `docs`, `test`, `fix`, `feat`.  
  _Example:_  
  ```
  feat(core): add new scheduler for task management
  fix(internal): correct off-by-one error in parser
  ```

## Workflows

### openspec-proposal-lifecycle

**Trigger:** When introducing, updating, auditing, or archiving an OpenSpec proposal or capability spec  
**Command:** `/openspec-proposal`

1. Create or update proposal and design documents under `openspec/changes/<proposal-name>/`
2. Add or update capability specs in `openspec/changes/<proposal-name>/specs/`
3. Update `tasks.md` to reflect progress or new tasks
4. Audit status of in-flight proposals, marking them as landed, active, or archived
5. Archive completed/inactive proposals to `openspec/changes/archive/` and promote stable specs to `openspec/specs/`
6. Update references in `CONTRIBUTING.md` or related documentation

_Example directory structure:_
```
openspec/
  changes/
    my-feature/
      proposal.md
      design.md
      specs/
        capabilityA.md
      tasks.md
  changes/archive/
    old-feature/
      proposal.md
  specs/
    stable-capability.md
CONTRIBUTING.md
```

### feature-or-bugfix-with-test

**Trigger:** When adding a new feature or fixing a bug in core or internal modules  
**Command:** `/feature-with-test`

1. Modify or add implementation code in `core/` or `internal/` directories
2. Add or update corresponding `*_test.go` files for the affected module
3. Commit both implementation and test changes together

_Example:_
```go
// core/scheduler/scheduler.go
func NewScheduler() *Scheduler {
    // implementation
}

// core/scheduler/scheduler_test.go
func TestNewScheduler(t *testing.T) {
    // test implementation
}
```

### scaffold-and-extend-e2e-test-module

**Trigger:** When introducing or expanding E2E test coverage for behavioral invariants  
**Command:** `/scaffold-e2e-test`

1. Create or update scaffolding files for the E2E test module (`go.mod`, `README.md`, `doc.go`, `.gitignore`)
2. Add new helpers or utilities under `tests/e2e/helpers/`
3. Write or extend `*_test.go` files to cover new helpers or behaviors
4. Update `openspec/changes/<proposal>/tasks.md` to track progress

_Example:_
```
tests/
  e2e/
    go.mod
    README.md
    doc.go
    .gitignore
    helpers/
      setup.go
      setup_test.go
openspec/
  changes/
    my-feature/
      tasks.md
```

## Testing Patterns

- **Test Framework:**  
  Standard Go testing (`testing` package); specific framework not detected.
- **Test File Naming:**  
  Suffix test files with `_test.go` and place them alongside the code under test.
  _Example:_  
  ```
  core/utils/math.go
  core/utils/math_test.go
  ```
- **Test Structure:**  
  Use named exports for test functions.  
  _Example:_  
  ```go
  func TestAddNumbers(t *testing.T) {
      // test logic
  }
  ```

## Commands

| Command                | Purpose                                                        |
|------------------------|----------------------------------------------------------------|
| /openspec-proposal     | Manage the lifecycle of OpenSpec proposals and specs           |
| /feature-with-test     | Add a new feature or bugfix with corresponding unit tests      |
| /scaffold-e2e-test     | Scaffold or extend the E2E test module and helpers            |
```