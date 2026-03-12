# Contributing to ROSA
Welcome, and thank you for considering to contribute to ROSA.
Before you begin, or have more questions reach out to us on [Slack @rosa-cli](https://redhat.enterprise.slack.com/archives/CB53T9ZHQ)

## Contributing Code
To contribute bug fixes or features to ROSA:

- Communicate your intent.
- BEFORE YOUR FIRST COMMIT IN A NEW CLONE, YOU MUST RUN `make install-hooks`.
- Make your changes.
- Test your changes.
- Run `make fmt` to align with the project formatting.
- Open a Pull Request (PR).

Communicate your intent in the form of a JIRA ticket on the [OCM](https://issues.redhat.com/projects/OCM) project.
To ensure it is picked up by the ROSA team, please set `component = rosa` in the ticket. All JIRA's are refined by the team on a weekly cadence.

Be sure to practice good git commit hygiene as you make your changes. All but the smallest changes should be broken up
into a few commits that tell a story. Use your git commits to provide context for the folks who will review PRs. We strive
to follow [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/#summary).

REQUIRED BEFORE YOUR FIRST COMMIT IN A CLONE:
```shell
make install-hooks
```

YOU MUST LET THE LOCAL HOOKS RUN ON EVERY COMMIT AND PUSH. DO NOT BYPASS LOCAL HOOKS.

The hooks perform:
- `pre-commit`: formats staged Go files (imports + gofmt) and re-stages the formatted files
- `commit-msg`: validates the commit message format
- `pre-push`: runs format-check, build, lint, changed-files coverage, and unit/integration tests
- `pre-push` runs against committed content and blocks when staged/unstaged tracked changes are present
- Prow re-runs these checks as required presubmits, so merges are blocked until they pass
- check runs are fail-fast: execution stops at the first failing step
- if you hit any bumps when committing, please let us know

Use this aggregated command before pushing:
```shell
make basic-checks                 # format + format-check + build + lint + changed-files coverage + unit/integration tests
```

`basic-checks` forces fresh test execution (no Go test cache) for test steps.
Coverage in these check flows evaluates changed Go files only and requires a minimum of 80% of executable changed lines.

Use these stage-specific commands when you want to run what each hook runs:
```shell
make pre-commit-checks
make pre-push-checks
```

Use these commands for individual checks:
```shell
make fmt-check
make rosa
make lint
make coverage-changed-files
make test
```

Commit message checks are performed by the `commit-msg` hook during commits.

Formatting helpers:
```shell
make fmt         # formats all Go files under cmd/, pkg/, tests/
make fmt-staged  # formats only staged Go files and re-stages them (used by pre-commit hook)
make fmt-check   # verifies formatting without rewriting files
```

If you want to inspect planned execution without running checks:
```shell
make run-checks -- pre-push --list-steps
make run-checks -- pre-push --dry-run
make run-checks -- basic --list-steps
make run-checks -- basic --dry-run
```

Hook scripts are internal and should not be run manually.

The commit message should follow this template:
```shell
OCM-XXXXX | <type>[optional scope][!]: <description>

[optional BODY]

[optional FOOTER(s)]
```

For example:
```shell
OCM-6141 | feat: Allow longer cluster names up to 54 chars 

Also allow users to supply an optional domain-prefix to customize the DNS

Signed-off-by: Foo Bar <foo.bar@baz.com>
```

The commit contains the following structural types, to communicate your intent:

- `fix:` a commit of the type fix patches a bug in your codebase (this correlates with PATCH in Semantic Versioning).
- `feat:` a commit of the type feat introduces a new feature to the codebase (this correlates with MINOR in Semantic
  Versioning).

Types other than `fix:` and `feat:` are allowed:
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to our CI configuration files and scripts
- `docs`: Documentation only changes
- `perf`: A code change that improves performance
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- `test`: Adding missing tests or correcting existing tests

All code should be covered by tests. We use [Ginkgo](https://onsi.github.io/ginkgo/). Other third party testing package
will be rejected.

Once you made and tested your changes, create a pull request (PR). In the PR `overview` please link the
jira ticket associated with your change. This should follow the format `JIRA: SDA-xxxx`. Note the key word `JIRA`,
use of any other key word may result in the bot performing unwanted action to the ticket in JIRA. Please also include in the
`overview` any additional information not in the JIRA that may help set context around your intent. Also include any extra
validation steps which may help reviews to validate the changes.

We work on a Sprint basis, all changes should be tracked by a JIRA. When being worked on these changes should be added to
the current SDA sprint. These sprints are denoted as `SDA - Sprint xxx`. The workflow should be as follows:
- `Todo` will complete in this current sprint
- `In Progress` ticket is currently being worked on
- `Code Review` PR has been created and is being reviewed by the team.
- `Review` Once the changes have been merged, move ticket to `Review` a QE person will be assigned to the ticket and tested.
- `Done` Once QE are satisfied with the change and bugs have been fixed the QE person assigned to your ticket will mark it
  as done

During `Review`, remain assigned to the ticket, so that QE knows who to assign any follow-up bugs to during testing. You
will also be asked to review test cases for this change. QE will supply you with a link to the test cases, if you approve
the test cases add `tc-approved` label to the JIRA, if you need changes to the test cases work with QE in the JIRA comments
to resolve these changes.

# CI
## Prow

This repository is using Prow CI running at https://prow.ci.openshift.org/,
configured in https://github.com/openshift/release repo.

`.golangciversion` file is read by the `lint` job commands there:
https://github.com/openshift/release/blob/master/ci-operator/config/openshift/rosa/openshift-rosa-master.yaml

# Style Guide

## Adding a New Command

### Add your Command to expected CLI Structure

We automatically test the structure of the ROSA CLI to ensure commands and command flags are not accidentally added or removed.
When you first create a new command, the test suite will fail because of this.

You need to add your command to the following file [command_structure](cmd/rosa/structure_test/command_structure.yml) in the correct
location within the command tree in order for this test to pass.

You additionally need to create a directory under the [command_args](cmd/rosa/structure_test/command_args) sub-directory 
and create a file called `command_args.yml`. This file should contain a simple yaml list of the `flags` supported by your command.
For example, a command with flag `foo`, `bar`, `bob` would have the following `command_args.yml`:

```yaml
- name: foo
- name: bar
- name: bob
```

## Error Handling in Commands

If you are contributing code, please ensure that you are handling errors properly. You should
not call `os.Exit()` in your Command (there is a significant amount of this in our code which we
are working to remove)

Please use `Run: run` instead of `RunE: runE` when writing commands,
   in order to stop the **usage info** being printed when an error is returned.

## Version-gating a feature

In some cases new features have minimal OCP versions.
To add validation for a minimal version, please add the minimal version
const to `features.go` and use the `IsFeatureSupported` function.

# Questions?

If you have any questions about the code or how to contribute, don't hesitate to
[open an issue](https://github.com/openshift/rosa/issues/new) in this repo
