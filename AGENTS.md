# Agent guide for openshift/rosa

This repository contains the public-facing Go CLI for managing Red Hat
OpenShift Service on AWS (ROSA). Start here, then read the documents that
match the change. Shared guidance is tool agnostic and lives in `guidelines/`.
`CLAUDE.md` and `GEMINI.md` point here.

| When working on | Read |
|-----------------|------|
| Repository layout, CLI/core boundaries, or migration | [Architecture](guidelines/ARCHITECTURE.md); [package classification](guidelines/refactor/pkg-architecture.md) for `pkg/` refactors |
| Cobra commands, flags, prompts, output, or Platform API dispatch | [Command guidelines](guidelines/command-guidelines.md) |
| Reusable `pkg/` workflows, Request/Result types, or validation | [Workflow conventions](guidelines/workflow-conventions.md) |
| Error creation, wrapping, translation, or exit behavior | [Error conventions](guidelines/error-conventions.md) |
| AWS-facing code, prerequisites, or product-facing setup text | [AWS guidelines](guidelines/aws-guidelines.md) |
| Go code and review | [Go guidelines](guidelines/go-guidelines.md) |
| Tests, generated files, command structure, or verification | [Testing guidelines](guidelines/testing-guidelines.md) |
| Credentials, secrets, or scanning | [Security](guidelines/security.md) |
| CI images, Go version alignment, or release boundaries | [Build and CI](guidelines/build-and-ci.md) |
| Compatibility, sensitive changes, or human review | [Change review](guidelines/change-review.md) |
| Project overview, installation, or user setup | [README](README.md) |
| Hooks, exact commands, commits, and contributor workflow | [Contributing](CONTRIBUTING.md), [Makefile](Makefile), and the referenced hook scripts |
| Preparing a PR | [PR template](.github/pull_request_template.md) |

The exact command source, template, hook script, structure-test contract, or
generator referenced by these documents is authoritative for its behavior.
Use `CONTRIBUTING.md` for contributor procedures and the PR template for
submission checks. The topic guides hold repository-specific rules.
