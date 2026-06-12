# ROSA CLI interactive prompts: Survey → Bubble Tea evaluation

**Jira:** ROSAENG-4069  
**Branch:** `ROSAENG-4069-bubble-tea`  
**Audience:** Product management  
**Status:** Spike complete on branch; **no migration decision yet**

---

## Why the ROSA CLI uses Survey today

- **Survey** (`github.com/AlecAivazis/survey/v2`) is a Go library for **interactive terminal prompts**: text input, single select, multi-select, yes/no confirm, password fields.
- ROSA uses it when a command runs in **interactive mode** (`-i` / `--interactive`) or when required flags are missing and the CLI prompts instead of failing.
- Survey handles **presentation and input only**. It is responsible for:
  - Showing the question, default, and optional help
  - Reading keyboard input in the terminal
  - Returning validation errors on invalid input
  - Stepping through questions one at a time
- ROSA business logic (OCM, AWS, validation rules) lives elsewhere. Survey is wrapped in `pkg/interactive/` and `pkg/interactive/confirm/`; commands call helpers such as `GetString`, `GetOption`, `GetMultipleOptions`, and `confirm.Prompt`.
- Usage is **broad**: simple ARN/prefix prompts through long wizards (cluster create, machine pools, IDP setup, upgrades). The branch inventory is in `developer-docs/cli-paths.md` (~157 command paths).

---

## Why staying on Survey is risky

- Survey is **archived and unmaintained** (repository archived April 2024). The author states they cannot continue maintenance and points adopters to alternatives, including Bubble Tea.
- ROSA pins **`v2.2.15`**. There is no supported path for bug fixes, security updates, or terminal/Go compatibility work.
- Open issues in the archived repo will not be resolved upstream.
- Impact of staying:
  - **Technical debt** on a core, user-facing layer
  - Higher risk when terminals or platforms change
  - No community or vendor support for interactive UX defects
- Staying on Survey is a **near-term default**, not a viable long-term strategy.

---

## Why we evaluated Bubble Tea (spike on `ROSAENG-4069-bubble-tea`)

**Bubble Tea** (`github.com/charmbracelet/bubbletea`) is a Go framework for terminal UIs. **Bubbles** provides reusable widgets (text input, lists, spinners). The Survey author and much of the Go CLI ecosystem treat this stack as the practical successor.

**Reasons for the spike:**

- **Actively maintained** — Charm Bracelet ecosystem with ongoing releases and adoption.
- **Aligned with industry direction** — common choice for modern terminal CLIs; Survey’s own README recommends it.
- **Better base for complex flows** — multi-step wizards, showing prior answers, and (with design work) editing earlier steps are explicit model concerns, not ad-hoc `AskOne` chains.
- **Testability** — wizard logic can be covered by automated tests (step order, validation, branching). ROSA has **no automated interactive flow tests** for Survey today; behavior is validated manually.
- **Extensibility** — custom UI when Bubbles has no drop-in widget (required for some ROSA prompt types).

POC commands on the branch mirror production flows (same questions, validators, outcomes) without calling AWS/OCM — they print a fake success for safe demo and review.

---

## Main concerns and trade-offs

### User experience will change

- Survey: compact, **line-oriented** prompts; answers accumulate in terminal scrollback.
- Bubble Tea: **TUI redraw** (multi-line blocks, list panels, styled inputs). Look and feel differ even when questions and outcomes match.
- No drop-in match for some Survey patterns (`?` in-prompt help, cyan instruction blocks in IDP/upgrade flows). These need redesign.
- On the branch, prior answers stay visible in the wizard summary (Survey-like intent); layout still differs from Survey.
- **Functional parity** is realistic; **identical UX** would need significant custom UI work.

### Migration is phased work, not a library swap

- Dozens of interactive call sites across the CLI.
- Per command: wizard model, non-interactive/flag path, structure tests, and tests.
- Suggested order: input / select / confirm first; cluster and machine pool last (highest complexity).

### Gaps requiring custom development

| Survey capability | Bubble Tea on branch |
|-------------------|----------------------|
| **MultiSelect** (subnets, security groups, tuning configs, …) | No Bubbles widget — **custom model needed** (implemented in machine pool demo; see golden-path workarounds table) |
| **Long single-select** (versions, regions) | `bubbles/list` + filter — **different UX**, often stronger; mostly native |
| **Confirm / yes-no** | `bubbles/list` or confirm model — **done in POC** |
| **PrintHelp** (instructions before prompts) | Not in POC — **must be built** |
| **`--color`** | Wired in machine pool transcript; not all wizards yet — **finish for parity** |
| **Survey scrollback Q&A** | Bubble Tea redraws one panel — machine pool demo uses native `tea.Println` for history (**light workaround**, not identical layout) |

**Machine pool spike takeaway:** single-select and text inputs map cleanly to Bubbles. **MultiSelect is the only hard gap** validated on the golden path (three prompts). That is expected engineering cost, not a blocker — budget custom widgets (or a shared `pkg/interactive/bubbletea/multiselect`) once, reuse across cluster/machine pool/IDP flows. Light workarounds (type-to-filter shim, transcript lines) are low risk and teatest-friendly.

### Testing

- Bubble Tea enables systematic tests; effort is **per wizard** (model flow tests + optional program smoke via experimental `teatest`).
- Still a clear improvement vs Survey (no interactive flow tests in ROSA today).

### Decision summary

| | Stay on Survey | Move to Bubble Tea |
|---|----------------|-------------------|
| Maintenance | None (archived) | Active ecosystem |
| User-visible UX | Unchanged | Will change |
| Engineering effort | Low near-term | High, phased |
| Interactive test coverage | Effectively none | Can be added |
| Complex commands (cluster, MP) | Dependency rot | Upfront cost; clearer long-term path |

---

## Demo

Short **GIF** of the POC on branch `ROSAENG-4069-bubble-tea`:

- **Command:** `rosa create user-role-bubble`
- **Flow:** role prefix → permissions boundary → role path → creation mode (auto / manual)
- **Highlights:** validation on bad input; prior answers remain visible while advancing; confirm step before fake auto-mode completion

**Demo recording:** *[insert GIF link or attachment here]*

**Try on branch:** `git checkout ROSAENG-4069-bubble-tea` → `make rosa` → `./rosa create user-role-bubble`

**Further reading (same branch):** `developer-docs/survey-bubbletea-migration.md`, `developer-docs/survey-bubbletea-migration-testing.md`

---

*Prepared from ROSAENG-4069 spike work on `ROSAENG-4069-bubble-tea`. Informational only — not a go/no-go recommendation.*
