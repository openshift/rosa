# AWS Guidelines

## Scope

Use this file when work touches `pkg/aws/`, AWS-backed command flows, setup instructions, or user-facing troubleshooting involving AWS prerequisites.

## Architecture And Prerequisites

- Check whether the behavior is specific to ROSA classic, ROSA with HCP, or shared before changing code or docs.
- Cross-check STS, IAM, OIDC, VPC, subnet, PrivateLink, quota, region, and credential assumptions against the official ROSA and AWS docs linked from `AGENTS.md`.
- When examples mention AWS CLI install, profiles, or config files, verify them against current AWS CLI documentation before editing.

## Implementation Rules

- Prefer the existing AWS client wrappers, helpers, and mocks already used in the surrounding package.
- Do not introduce raw credentials into code, tests, logs, examples, or issue templates.
- Treat account-setup and prerequisite messaging as product behavior: do not invent requirements when the docs already define them.

## Dependency Guardrails

- Do not silently bump `aws-sdk-go-v2` or related AWS dependencies as part of an unrelated change.
- If an AWS dependency bump is required, call it out explicitly in the commit and PR, explain why it is needed, and validate the downstream impact.
- Avoid `go mod tidy` or `go.mod`/`go.sum` churn unless the task explicitly requires dependency-state changes.

## Review Prompts

- Does this change preserve the distinction between HCP and classic where it matters?
- Are quota, SCP, IAM, STS, or OIDC assumptions backed by current official docs?
- Would a dependency or example change affect user setup, release behavior, or support expectations outside the edited file?
