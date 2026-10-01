---
name: ROSA AWS Context
description: "Cross-check AWS-facing code and docs against official ROSA and AWS references before changing behavior."
---

# ROSA AWS Context

Use this skill for AWS-facing code, setup flows, architecture wording,
prerequisites, or troubleshooting.

1. Read [AGENTS.md](../../../AGENTS.md) and its
   [AWS guidelines](../../../guidelines/aws-guidelines.md).
2. Identify whether the behavior applies to HCP, classic, or both. Check the
   relevant official sources linked by the AWS guide.
3. Apply the guide's implementation, credential, and dependency rules. Surface
   any disagreement between code and official documentation.
4. Re-read the source for each changed claim and check user-facing examples.
5. Select the relevant local checks from
   [CONTRIBUTING.md](../../../CONTRIBUTING.md).
