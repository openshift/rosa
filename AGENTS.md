# AGENTS.md

This is the central repo-local guidance for agents working inside `openshift/rosa`.

Use this file as the starting point for repository context. When this file points to an exact command source, template, hook script, or generated-file workflow, treat that referenced file as the authoritative procedure.

## Repository Scope

- `openshift/rosa` is the ROSA CLI: a public-facing Go CLI for managing Red Hat OpenShift Service on AWS (ROSA).
- This repository ships frequently and supports real customer workflows. Be conservative. Small mistakes can break release builds, presubmits, E2E flows, or user-facing command behavior.
- The human submitter owns every change. Agents are helpers, not decision makers.
- Do not perform release work or release automation from an agent session.
- Do not duplicate code from the Go standard library or vendored dependencies when an existing implementation already covers the need.

## Key Reference Files

- `CONTRIBUTING.md`
  - Contributor workflow, hook installation, required local checks, commit format, and CI expectations.
- `Makefile`
  - Supported local build, test, format, and generation commands.
- `.github/pull_request_template.md`
  - Required PR structure, validation notes, and reviewer-facing checklist.
- `.githooks/pre-commit`, `.githooks/pre-push`, `.githooks/commit-msg`
  - Hook entrypoints used by local git workflow.
- `hack/pre-commit-hook.sh`, `hack/pre-push-hook.sh`, `hack/commit-msg-hook.sh`, `hack/commit-msg-verify.sh`
  - Exact local validation and commit-message behavior.
- `cmd/rosa/structure_test/command_structure.yml`
  - CLI command tree contract.
- `cmd/rosa/structure_test/command_args/**/command_args.yml`
  - Supported flag contract for each command.

## Docs Index

- `guidelines/ARCHITECTURE.md`
  - High-level CLI layering, external-system boundaries, and change-risk hotspots.
- `guidelines/aws-guidelines.md`
  - AWS and ROSA architecture checks, prerequisite handling, secret safety, and dependency-bump guardrails.
- `guidelines/command-guidelines.md`
  - CLI command authoring expectations, structure-test alignment, and reporter/output patterns.
- `guidelines/testing-guidelines.md`
  - Test style, generated-file boundaries, validation paths, and PR-readiness checklist use.

## Codebase Map

- `cmd/rosa/`
  - Cobra commands and CLI wiring.
- `pkg/`
  - Business logic, services, helpers, AWS integration, OCM integration, output, and interaction layers.
- `pkg/aws/`
  - AWS SDK integrations plus interfaces and generated mocks.
- `pkg/ocm/`
  - OpenShift Cluster Manager API integrations.
- `pkg/output/`, `pkg/reporter/`, `pkg/interactive/`
  - User-facing messages, reporter behavior, and interactive flows.
- `tests/e2e/`
  - End-to-end tests and environment-dependent coverage.
- `cmd/docs/`
  - CLI documentation generation.
- `templates/`
  - CloudFormation and other templated assets.
- `assets/bindata.go`
  - Generated asset file. Do not edit by hand.
- `pkg/*/mocks/`, `cmd/create/idp/mocks/`
  - Generated mocks. Do not edit by hand.
- `vendor/`
  - Vendored dependencies. Do not edit directly.

## Working Rules

- Keep Cobra-specific logic in `cmd/`; keep non-Cobra logic in `pkg/`.
- Use existing package patterns before inventing new abstractions. The machinepool commands are a strong reference for newer command structure.
- Prefer existing helpers and functions when they already fit the task.
- Follow the command entrypoint and exit pattern already established in the nearest comparable command area.
- Many ROSA commands use `Run: run`; do not switch a command area between `Run` and `RunE`, or add/remove direct `os.Exit()` calls, unless the surrounding pattern already does so and the change keeps behavior consistent.
- Keep error handling consistent with the surrounding package, especially reporter usage, exit behavior, and wrapped error messages.
- Follow repo naming conventions, including the existing acronym style such as `variableNameEndingWithAcronymHcp`.
- Keep variable names explicit and consistent with nearby code.
- Respect generated-file boundaries. If a change requires regenerating mocks or assets, use the documented generator path instead of hand-editing output.

## Command Authoring Rules

When adding or changing a CLI command:

- Update `cmd/rosa/structure_test/command_structure.yml` when the command tree changes.
- Update or add the matching `cmd/rosa/structure_test/command_args/.../command_args.yml` file when supported flags change.
- Keep user interaction, prompting, and display logic aligned with existing `output`, `interactive`, and reporter patterns.
- Prefer placing business logic in `pkg/` and keeping Cobra command files thin.
- Review similar commands in the same area before adding new flags, validation, or flow control.

## Tests And Verification

- Run `make install-hooks` once per clone before committing.
- Do not bypass local hooks.
- Common checks:
  - `make fmt`
  - `make fmt-check`
  - `make lint`
  - `make test`
  - `make basic-checks`
  - `make pre-push-checks`
  - `make rosa`
  - `make generate`
  - `make generate-docs`
- Use the developer checklist in `.github/pull_request_template.md` as the final PR-readiness pass for validation, manual testing, docs updates, and risk notes.
- Add focused automated tests when behavior changes in a way that could regress.
- Do not change tests to accommodate broken behavior. Tests should prove correctness, not hide regressions.
- Use Ginkgo v2 and Gomega in the style already used by the surrounding package.
- Generated mocks must come from `make generate`, not manual edits.
- Do not run `go mod tidy`, `go mod vendor`, or `make verify` unless the task explicitly requires dependency-state changes or the user asked for that workflow. `make verify` rewrites dependency state.

## AWS And Product Truth Sources

When work touches AWS-facing behavior, user-facing docs, setup instructions, or architecture wording, prefer official product docs over memory.

Use these sources first:

- [Red Hat ROSA documentation](https://docs.redhat.com/en/documentation/red_hat_openshift_service_on_aws/4/html/about/welcome-index)
- [AWS ROSA architecture](https://docs.aws.amazon.com/rosa/latest/userguide/rosa-architecture-models.html)
- [Set up to use ROSA](https://docs.aws.amazon.com/rosa/latest/userguide/set-up.html)
- [Create a ROSA with HCP cluster using the ROSA CLI](https://docs.aws.amazon.com/rosa/latest/userguide/getting-started-hcp.html)
- [AWS CLI install](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- [AWS CLI configuration files](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)

Cross-check the docs when the change involves:

- HCP versus classic architecture wording or behavior
- STS, IAM, OIDC, or AWS account prerequisites
- VPC, subnet, PrivateLink, DNS, or quota assumptions
- AWS CLI installation, profiles, credentials, or config examples
- User-facing setup or troubleshooting guidance

- Do not invent AWS or ROSA product behavior when the official docs already define it.
- Do not silently bump AWS SDK, OCM SDK, or related dependency versions. If a dependency bump is required, call it out explicitly, explain why, and validate downstream impact.

## Commit And PR Expectations

- Commit subject format:
  - `OCM-XXXXX | <type>[optional scope][!]: <description>` or `ROSAENG-XXXX | <type>[optional scope][!]: <description>`
- Allowed types include:
  - `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `build`, `ci`, `perf`
- Use `.github/pull_request_template.md` for every PR.
- Explain both what changed and why it changed.
- Link Jira and any related PRs or docs.
- Include reproducible validation steps and call out any risk, limitation, or follow-up work.

## Agent-Facing File Layout

- `AGENTS.md`
  - Central repo-local guidance for agents in this repository.
- `CONTRIBUTING.md`
  - Canonical contributor workflow and validation reference for humans and agents.
- `CLAUDE.md`
  - Thin Claude-specific entrypoint that imports this file.
- `GEMINI.md`
  - Thin Gemini-specific entrypoint that points back here.
- `guidelines/ARCHITECTURE.md`
  - Repository architecture context that should stay stable across agent tools.
- `guidelines/*-guidelines.md`
  - Domain-specific guidance that `AGENTS.md` indexes instead of duplicating.
- `.cursor/rules/`
  - Tool-specific reinforcement of the rules in this file.
- `.claude/skills/`
  - Small ROSA-specific workflows that package repeatable tasks without replacing this file.

## Human-In-The-Loop Triggers

- A change wants to bump `aws-sdk-go-v2`, `ocm-sdk-go`, Cobra, Ginkgo, or otherwise requires `go mod tidy`, `go mod vendor`, or broad dependency rewrites.
- A new feature or command flow appears to duplicate an existing ROSA workflow, Jira ticket, or merged PR and the intended scope is unclear.
- The change touches login, authentication, token storage, keyrings, credentials, STS, IAM, OIDC, break-glass, or other security-sensitive paths.
- The change alters command structure, flags, prompts, JSON output, or user-facing setup behavior in a way that may affect backward compatibility.
- Generated files, structure tests, or broader test expectations change in ways that are larger than the task appears to justify.

## Safety Reminders

- Be cautious with monthly-release codepaths and user-facing commands.
- Be cautious when touching AWS and OCM integration flows.
- Do not edit generated files directly.
- Do not weaken tests.
- Do not perform release work from an agent session.
