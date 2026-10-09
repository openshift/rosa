# AWS Guidelines

## Scope

Use this file when work touches `pkg/aws/`, AWS-backed command flows, setup instructions, or user-facing troubleshooting involving AWS prerequisites.

Official product and setup references:

- [Red Hat ROSA documentation](https://docs.redhat.com/en/documentation/red_hat_openshift_service_on_aws/4/html/about/welcome-index)
- [AWS ROSA architecture](https://docs.aws.amazon.com/rosa/latest/userguide/rosa-architecture-models.html)
- [Set up to use ROSA](https://docs.aws.amazon.com/rosa/latest/userguide/set-up.html)
- [Create a ROSA with HCP cluster using the ROSA CLI](https://docs.aws.amazon.com/rosa/latest/userguide/getting-started-hcp.html)
- [AWS CLI install](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- [AWS CLI configuration files](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)

## Architecture And Prerequisites

- Check whether the behavior is specific to ROSA classic, ROSA with HCP, or shared before changing code or docs.
- Cross-check STS, IAM, OIDC, AWS account, VPC, subnet, PrivateLink, DNS,
  quota, region, and credential assumptions against the official sources above.
- When examples mention AWS CLI install, profiles, or config files, verify them against current AWS CLI documentation before editing.
- Confirm setup claims, including quotas, support plans, SCP constraints, and
  STS token version notes, against the official setup documentation.

## Implementation Rules

- Prefer the existing AWS client wrappers, helpers, and mocks already used in the surrounding package.
- Do not introduce raw credentials into code, tests, logs, examples, or issue templates.
- Treat account-setup and prerequisite messaging as product behavior: do not invent requirements when the docs already define them.
- Keep OCM-facing behavior aligned with the client and output patterns under
  `pkg/ocm/`.
- If code behavior and official docs disagree, surface the mismatch instead
  of guessing.

## Dependency Guardrails

- Do not silently bump `aws-sdk-go-v2`, `ocm-sdk-go`, or related dependencies
  as part of an unrelated change.
- If an AWS, OCM, or related dependency bump is required, call it out
  explicitly in the commit and PR, explain why it is needed, and validate
  downstream impact.
- Avoid `go mod tidy` or vendor churn unless the task explicitly requires dependency-state changes.

## Review Prompts

- Does this change preserve the distinction between HCP and classic where it matters?
- Are quota, SCP, IAM, STS, or OIDC assumptions backed by current official docs?
- Would a dependency or example change affect user setup, release behavior, or support expectations outside the edited file?
