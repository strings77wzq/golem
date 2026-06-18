// E2E test module for Golem.
//
// This module is intentionally separate from the main project module
// (github.com/strings77wzq/golem) so that end-to-end tests can only
// observe Golem through its public binary surface, never by importing
// internal Go packages. See specs/e2e-test-harness/spec.md, requirement
// "Black-box test execution against the real Golem binary".
//
// Dependencies are restricted to the standard library to keep the
// black-box property strict and to avoid polluting the main go.sum.
module github.com/strings77wzq/golem/tests/e2e

go 1.25.0
