# Agent Instructions

This file contains repository-specific instructions for AI coding agents.

## Required Context

- Read `README.md` for the project's purpose, supported formats, CLI behavior,
  and user configuration.
- Read `CONTRIBUTING.md` for development commands, tooling, changelog handling,
  and the release process.
- Treat those files as the source of truth for human-facing information. Do not
  duplicate their content here. Update the appropriate human-facing document
  when a change makes it inaccurate.

## Repository Boundaries

- `pkg/parser` is the public parsing API. Consider compatibility before changing
  exported types or behavior.
- `internal/pkg/settings` and `internal/pkg/batchconvert` are internal
  implementation packages.
- `cmd/go-homebank-csv` is the CLI entry point.

## Parser Changes

- Treat existing headers, encodings, date formats, amount formats, and fixture
  output as intentional unless a concrete export sample or requirement shows
  otherwise.
- Keep parser changes scoped to the affected source format. Do not make other
  parsers more permissive as an incidental cleanup.
- Back parser behavior changes with representative fixtures and tests. Cover
  successful input and relevant malformed input, including `ParserError` type,
  field, and line information where applicable.
- Use only synthetic, sanitized financial data in fixtures. Never commit real
  names, account details, transaction data, credentials, or unsanitized bank
  exports.

## Change Discipline

- Preserve unrelated worktree changes and avoid opportunistic refactoring.
- Prefer the smallest change that solves the stated problem and follows the
  surrounding code style.
- Do not commit generated binaries, release artifacts, OS/editor metadata, or
  ad hoc conversion output. Expected test fixtures are intentional artifacts
  and may be updated when behavior changes.
- Avoid new dependencies when the Go standard library reasonably covers the
  requirement. Treat dependency and Go version changes as explicit work.

## Verification

- Use focused package tests while iterating, then run the project checks
  documented in `CONTRIBUTING.md` before completing code changes.
- Report checks that could not be run and the reason. Documentation-only
  changes do not require the Go test suite unless they affect executable
  examples or the user requests it.

## Changelog

- Follow the Changie workflow documented in `CONTRIBUTING.md`; do not edit the
  generated `CHANGELOG.md` directly outside that workflow.
- For every Dependabot update, manually add an unreleased Changie entry in the
  `Fixed` category. This applies to Go modules, GitHub Actions, and other
  Dependabot-managed dependencies.
- Match the established wording for dependency entries, for example:
  `Bump <dependency> from <old version> to <new version>`.
- Do not assume Dependabot creates the changelog entry; the human or agent
  applying the update is responsible for it.
