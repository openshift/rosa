# Security

Router: [`AGENTS.md`](../AGENTS.md). AWS and credential handling: [`aws-guidelines.md`](aws-guidelines.md).

## Secrets

When editing CLI code, tests, docs, examples, or logs:

- MUST NOT: Hard code secrets, API keys, tokens, kubeconfigs, AWS credentials, or customer identifiers.
- MUST NOT: Log or print credentials, tokens, or other secrets.
- DEFAULT: Prefer placeholders and variables in examples and test fixtures.

## Gitleaks (secret scanning)

- MUST: Run `make verify-gitleaks` (also part of `make pre-push-checks` / Prow `ci/prow/pre-push-checks`) and the blocking pre-commit gitleaks gate (`make pre-commit-checks` / `.githooks/pre-commit`).
- MUST NOT: Bypass with `SKIP=gitleaks` or `git commit --no-verify`.
- MUST: Prefer fixing findings. MAY allowlist only justified mocks/fixtures in `.gitleaks.toml` with a short comment — never disable the scan.
- Config: `.gitleaks.toml`. Makefile pin: `GITLEAKS_VERSION` (release tag). Pre-commit pin: commit SHA with `# frozen: <same tag>` in `.pre-commit-config.yaml`.
- Commands and Renovate notes: [`CONTRIBUTING.md`](../CONTRIBUTING.md).
