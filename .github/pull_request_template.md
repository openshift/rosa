<!--
Please provide enough context so reviewers can understand:
1) the problem,
2) why this change is needed,
3) what changed,
4) how you validated it.

Use N/A when the option is not applicable to your case.

Commit format requirement:
[JIRA-TICKET] | [TYPE]: <MESSAGE>
Supported ticket prefixes:
OCM-XXXXX, ROSAENG-XXXX
TYPE must be one of:
feat, fix, docs, style, refactor, test, chore, build, ci, perf
For contributor workflow, see: ./CONTRIBUTING.md
For repo-local agent guidance, see: ./AGENTS.md
-->

## PR Summary
<!-- brief text of 1 or 2 lines with the most important changes and outcomes -->

## Detailed Description of the Issue
<!-- Describe the root problem, scope, impact, and user/business context -->

## Related Issues and PRs
<!-- Link all tracking items and related code changes -->
- Jira: [OCM-XXXXX](https://issues.redhat.com/browse/OCM-XXXXX) or [ROSAENG-XXXX](https://issues.redhat.com/browse/ROSAENG-XXXX)
- Fixes: `#`
- Related PR(s):
- Related design/docs:

## Type of Change
<!-- Check the primary type this PR represents -->
- [ ] feat - adds a new user-facing capability.
- [ ] fix - resolves an incorrect behavior or bug.
- [ ] docs - updates documentation only.
- [ ] style - formatting or naming changes with no logic impact.
- [ ] refactor - code restructuring with no behavior change.
- [ ] test - adds or updates tests only.
- [ ] chore - maintenance work (tooling, housekeeping, non-product code).
- [ ] build - changes build system, packaging, or dependencies for build output.
- [ ] ci - changes CI pipelines, jobs, or automation workflows.
- [ ] perf - improves performance without changing intended behavior.

## Previous Behavior
<!-- What users or systems experienced before this change -->

## Behavior After This Change
<!-- What changes now, including user-visible and non-user-visible behavior -->

## How to Test (Step-by-Step)
<!-- Provide reproducible validation instructions -->
### Preconditions
<!-- Required setup, environment, credentials, flags, cluster state, etc. -->

### Test Steps
1.
2.
3.

### Expected Results
<!-- What should happen after running the steps above -->

## Proof of the Fix
<!-- Attach evidence that demonstrates the changed behavior -->
- Screenshots:
- Videos:
- Logs/CLI output:
- Other artifacts:

## Breaking Changes
- [ ] No breaking changes
- [ ] Yes, this PR introduces a breaking change (describe impact and migration plan below)

### Breaking Change Details / Migration Plan
<!-- Required only when breaking changes are introduced -->

## Developer Verification Checklist
- [ ] Commit subject/title follows `[JIRA-TICKET] | [TYPE]: <MESSAGE>`.
- [ ] PR description clearly explains both **what** changed and **why**.
- [ ] Relevant Jira/GitHub issues and related PRs are linked.
- [ ] `make install-hooks` has been run in this clone.
- [ ] Tests were added/updated where appropriate.
- [ ] I manually tested the change.
- [ ] `make test` passes.
- [ ] `make lint` passes.
- [ ] `make rosa` passes.
- [ ] Documentation or repo-local agent guidance was added/updated where appropriate.
- [ ] Any risk, limitation, or follow-up work is documented.
