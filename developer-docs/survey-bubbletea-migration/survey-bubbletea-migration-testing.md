# Survey to Bubble Tea: interactive testing

Companion to [survey-bubbletea-migration.md](./survey-bubbletea-migration.md). Documents how ROSA tests interactive CLI flows today, what the Bubble Tea POC adds, and the trade-offs between model-level tests and full-program tests.

**Reference POC**

- Command: `rosa create user-role-bubble` (`cmd/create/userrolebubble/`)
- Wizard model: `pkg/interactive/bubbletea/userrole/wizard.go`
- Confirm model: `pkg/interactive/bubbletea/confirm.go`
- Shared non-interactive helpers: `pkg/userrole/create.go` (`pkg/userrole/create_test.go`)

---

## Survey today: no interactive flow tests

ROSA’s Survey usage lives in `pkg/interactive/` and `pkg/interactive/confirm/`. Commands call helpers such as `GetString`, `GetOption`, and `confirm.Prompt`; they do not import Survey directly in most cases.

**What is tested**

| Area | Coverage |
|------|----------|
| Validators (`RegExp`, `ARNValidator`, `IsCIDR`, etc.) | Yes — `pkg/interactive/validation_test.go` exercises validator functions via Survey’s `core` types |
| Non-interactive command logic (flags, API calls, file output) | Yes — per-package unit tests where helpers exist |
| Survey prompt flow (step order, Enter/Esc, defaults, error messages on screen) | **No** — there are no tests that drive Survey prompts or assert terminal output |
| End-to-end CLI with a pseudo-TTY | **No** — no automated “type answers into the wizard” coverage for Survey commands |

Survey runs as a blocking, stdin-driven library call inside real commands. Without a TTY harness or refactor to expose a testable model, prompt behavior is validated manually only.

**Implication for migration:** moving to Bubble Tea does not remove existing test coverage for interactive flows, because that coverage does not exist for Survey. The POC establishes patterns we can reuse when migrating other commands.

---

## Bubble Tea: model-level flow tests (recommended default)

Bubble Tea separates **model** (`Update`, `View`, `Init`) from **program** (`tea.NewProgram`). Tests can call `Update` directly with `tea.KeyMsg` values — fast, deterministic, no terminal emulator, no extra dependencies beyond Ginkgo/Gomega already used in ROSA.

**File:** `pkg/interactive/bubbletea/userrole/wizard_flow_test.go`

This file uses Ginkgo and drives the wizard (and confirm) models without `tea.NewProgram` or `teatest`.

| # | Scenario | What it asserts |
|---|----------|-----------------|
| 1 | Happy path step order | Prefix → permissions boundary → path → mode; final `userrole.Input` |
| 2 | Required prefix | Empty prefix stays on step, shows `role prefix is required` |
| 3 | Invalid boundary ARN | Stays on boundary step, validator error message |
| 4 | Empty optional boundary | Advances to role path |
| 5 | Mode branching | Auto vs manual selection recorded in result |
| 6 | Confirm step | `y` / `n` / Enter (default yes/no) / Esc (abort) on `NewConfirmModel` |

**Strengths**

- Runs in normal `go test` / `make test` with no TTY
- Tests business rules (validation, step transitions, result struct) directly
- Stable — not tied to terminal width, ANSI sequences, or Bubble Tea patch versions
- Matches how most Charm projects test non-trivial models

**Limits**

- Does not exercise `tea.NewProgram`, alt-screen setup, or signal handling
- Does not assert rendered terminal output byte-for-byte (only `View()` substrings where used)
- Confirm tests cover the Bubble Tea confirm model, not the Survey `confirm` package used elsewhere

**When to add more:** extend this file (or sibling `*_flow_test.go` files) when migrating a command — same numbered-comment style keeps scenarios easy to review in PRs.

---

## Bubble Tea: full-program tests with teatest (optional, experimental)

[teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest) (`github.com/charmbracelet/x/exp/teatest`) runs a real Bubble Tea program in tests: fixed terminal size, key input via `Send`, wait helpers on program output, and `FinalModel` after quit.

**File:** `pkg/interactive/bubbletea/userrole/wizard_teatest_test.go`

**Current scenario:** `TestWizardHappyPathTeatest` — accept defaults on all four wizard steps (Enter through prefix, boundary, path, mode) and assert the final `wizardModel` is done with expected `userrole.Input` fields.

**Strengths**

- Closest automated check to “user ran the program” without manual CLI runs
- Can wait for output containing prompt labels before sending keys
- Useful smoke test that wiring between model and program still works

**Risks and costs**

| Risk | Detail |
|------|--------|
| **Experimental API** | Lives under `github.com/charmbracelet/x/exp/…`. APIs and behavior can change without semver guarantees on `exp`. |
| **Dependency churn** | Pulling in teatest may transitively bump `bubbletea` and related Charm modules (see `go.mod`). Review version impact on migration PRs. |
| **Flaky output waits** | `WaitFor` on raw program output can break if labels, styling, or terminal size change — even when behavior is correct. |
| **Golden / snapshot drift** | teatest’s ecosystem includes golden-file helpers; broad golden tests of full TUI output are brittle in CI across OS and terminal profiles. |
| **Slower** | Full program lifecycle is heavier than direct `Update` calls. |

**Recommendation for ROSA**

1. Prefer **`wizard_flow_test.go`** for validation, branching, and step logic — this should be the bulk of interactive tests.
2. Keep **one or few teatest happy-path smokes** per migrated wizard (as in `wizard_teatest_test.go`) if the team accepts the `x/exp` dependency.
3. Avoid large golden snapshots of entire wizard screens unless there is a strong maintenance story.
4. Do **not** add teatest for Survey — it does not apply to Survey’s architecture.

---

## Running the POC tests

The POC has three test targets: shared command helpers (no TUI), wizard flow tests (model-level), and one teatest smoke (full program).

**`pkg/userrole/`** — `create_test.go` covers non-interactive logic used by both `create user-role` and `create user-role-bubble`: input validation (prefix rules, optional fields), role name/ARN derivation, and manual-mode command strings. No Bubble Tea or Survey involved.

**`pkg/interactive/bubbletea/userrole/`** — interactive wizard and confirm behavior for the Bubble Tea POC:

- `wizard_flow_test.go` (`TestUserRoleWizardFlow`, Ginkgo) — drives `wizardModel` and `NewConfirmModel` by calling `Update` with key messages. Covers step order, validation errors, mode selection, and yes/no confirm outcomes without starting `tea.NewProgram`.
- `wizard_teatest_test.go` (`TestWizardHappyPathTeatest`) — runs the real Bubble Tea program via teatest, sends Enter through all four wizard steps with defaults, and checks the final model’s `userrole.Input`.

```bash
go test ./pkg/interactive/bubbletea/userrole/ -count=1
go test ./pkg/userrole/ -count=1
```

To run only the teatest smoke: `go test ./pkg/interactive/bubbletea/userrole/ -count=1 -run TestWizardHappyPathTeatest`

---

## Maintenance

Update this document when:

- New Bubble Tea wizards gain flow or teatest files.
- teatest or `bubbletea` versions change in `go.mod`.
- ROSA adopts a different TTY/integration test strategy (e.g. scripted CLI E2E).
- Survey interactive testing is added (unlikely without a refactor) — note what was added and where.
