---
name: ROSA Verification Gates
description: "Choose the right local verification steps before claiming a ROSA change is complete."
---

# ROSA Verification Gates

Start with `AGENTS.md` and `guidelines/testing-guidelines.md`.

Use this skill when:

- Preparing to commit or open a PR
- Deciding which checks are appropriate for a change
- Verifying docs, command, or generated-file changes

## Verification Mapping

### Go code changes

- `make fmt`
- relevant package tests or `make test`
- `make lint`
- `make rosa`

### Command or flag changes

- all Go code checks above
- verify `cmd/rosa/structure_test/command_structure.yml` and matching `command_args.yml`
- `make generate-docs` when help text or docs changed

### Generated mocks or assets

- `make generate`
- relevant tests for the touched package or command

### Pre-push confidence

- `make basic-checks`
- `make pre-push-checks` before push when the change is ready
- re-read `.github/pull_request_template.md` and use its developer checklist as the final PR-readiness pass

## Rules

- Run `make install-hooks` in a new clone before first commit.
- Do not bypass hooks.
- Do not run `go mod tidy` unless the task explicitly requires dependency-state changes.
- If generated files changed unexpectedly, stop and confirm why before committing them.
- If a command changed but the structure-test files did not, verify that omission intentionally.
- If the PR template checklist expects docs, manual validation, or risk notes, make sure the final PR body actually covers them.
