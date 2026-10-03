# Change Review

Use this file when a change may affect compatibility, security-sensitive
behavior, generated contracts, or require a human decision. The human
submitter owns every change; agents are helpers, not decision makers.
This public CLI ships frequently; small changes can affect customer workflows,
release builds, presubmits, or E2E runs.

For the user-facing compatibility contract covering flags, output, prompts,
and exit behavior, see
[architecture compatibility rules](ARCHITECTURE.md#backward-compatibility).

Seek human direction when:

- A change requires a bump to `aws-sdk-go-v2`, `ocm-sdk-go`, Cobra, Ginkgo,
  `go mod tidy`, `go mod vendor`, or a broad dependency rewrite.
- A new feature or command appears to duplicate an existing ROSA workflow,
  Jira ticket, or merged PR and its intended scope is unclear.
- A change touches login, authentication, token storage, keyrings,
  credentials, STS, IAM, OIDC, break-glass, or another security-sensitive path.
- A change alters command structure, flags, prompts, JSON output, or user-facing
  setup behavior in a way that may affect backward compatibility.
- Generated files, structure tests, or broader test expectations change more
  than the task appears to justify.

Use the [PR template](../.github/pull_request_template.md) for submission.
Explain what changed and why, link Jira and related PRs or docs, include
reproducible validation, and call out risks, limitations, or follow-up work.
Keep issue and PR templates specific to real ROSA workflows and reproducible
reports.
When reviewing documentation, check for stale commands, file paths,
placeholder Jira links, and wording that disagrees with repository workflows.
Prefer concise, stable instructions over generic AI boilerplate.
