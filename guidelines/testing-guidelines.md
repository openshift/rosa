# Testing Guidelines

## Scope

Use this file when adding or changing unit tests, structure tests, generated files, or local validation expectations.

## Test Style

- Use Ginkgo v2 and Gomega in the style already established in the surrounding package.
- Add focused coverage for behavior changes, especially new flags, branching logic, and error paths.
- Do not weaken assertions or rewrite tests to hide broken behavior.

## Structure And Generated Files

- If command structure or supported flags change, update the structure-test YAML files under `cmd/rosa/structure_test/`.
- The command tree contract is `cmd/rosa/structure_test/command_structure.yml`;
  matching `cmd/rosa/structure_test/command_args/**/command_args.yml` files
  define supported flags.
- Generated mocks must come from `make generate`; do not hand-edit files under `pkg/*/mocks/` or `cmd/create/idp/mocks/`.
- Do not hand-edit generated `assets/bindata.go` or vendored dependencies in
  `vendor/`. Use the documented generator when regeneration is required.
- If generated files change unexpectedly, stop and confirm why before committing them.

## Validation Paths

Use [CONTRIBUTING.md](../CONTRIBUTING.md) for hook installation, the local
format, build, lint, test, and aggregation commands, and exact pre-push checks.
When help text or command docs change, include `make generate-docs` in the
validation path.

Use the change-specific verification mapping in `CONTRIBUTING.md`. If
command behavior changes but the structure-test files do not, verify that
the omission is intentional.

## Dependency State

- Do not run `go mod tidy`, `go mod vendor`, or `make verify` unless the task
  explicitly requires dependency-state changes or that workflow. `make verify`
  rewrites dependency state.

## PR Readiness

- Re-read `.github/pull_request_template.md` before pushing and use its developer checklist as the final PR-readiness pass.
- Make sure the PR body includes the validation steps you actually ran, plus any manual checks, docs updates, risks, or follow-up work that the checklist expects.

## Review Prompts

- Does this change need new or updated automated coverage?
- Did command or flag edits stay aligned with structure-test YAML and generated docs?
- Does the PR body match the validation that was actually run for the change?
