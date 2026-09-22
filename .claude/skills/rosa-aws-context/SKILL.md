---
name: ROSA AWS Context
description: "Cross-check AWS-facing code and docs against official ROSA and AWS references before changing behavior."
---

# ROSA AWS Context

Use this skill when:

- Touching `pkg/aws/` or AWS-related command flows
- Editing AWS setup, prerequisites, or troubleshooting docs
- Changing wording around HCP, classic, STS, IAM, OIDC, VPC, subnet, quota, or region behavior

## Workflow

1. Read `AGENTS.md` and `guidelines/aws-guidelines.md`, then use the linked external ROSA and AWS references first.
2. Determine whether the change is HCP-only, classic-only, or shared.
3. Cross-check architecture claims against the ROSA architecture docs before changing code comments, help text, or docs.
4. Confirm prerequisite claims against the ROSA setup docs, especially quotas, support plans, SCP constraints, and STS token version notes.
5. Validate AWS CLI install, profile, and config guidance against official AWS CLI documentation before editing examples.
6. Prefer existing AWS helper functions and client wrappers over ad-hoc SDK usage.
7. Do not silently bump AWS SDK or related dependency versions; if a bump is required, call it out explicitly, explain why, and validate downstream impact.
8. Do not hard code credentials or add logging that exposes secret material.
9. If code behavior and official docs appear to disagree, surface the mismatch explicitly instead of guessing.

## Verification

- Re-read the exact doc section that supports the changed behavior or wording.
- Check user-facing examples for current commands and options.
- Run the relevant local checks from `CONTRIBUTING.md` and `Makefile`.
