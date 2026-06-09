# Survey to Bubble Tea migration reference

Reference for contributors and agents replacing [Survey](https://github.com/AlecAivazis/survey/v2) (`github.com/AlecAivazis/survey/v2`) with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Bubbles](https://github.com/charmbracelet/bubbles) in the ROSA CLI.

**Related files**

- `pkg/interactive/interactive.go` — Survey wrappers used today
- `pkg/interactive/validation.go` — shared validators (`Validator` type is Survey-based)
- `pkg/interactive/confirm/confirm.go` — yes/no prompts via Survey
- `pkg/interactive/bubbletea/` — Bubble Tea POC helpers
- `cmd/create/userrolebubble/` — reference POC command (`rosa create user-role-bubble`)
- `docs/cli-paths.md` — interactive command inventory and complexity ranking
- `docs/survey-bubbletea-migration-testing.md` — Survey vs Bubble Tea interactive testing strategy

**Dependencies (POC branch)**

- `github.com/charmbracelet/bubbletea v1.1.0`
- `github.com/charmbracelet/bubbles v0.20.0`

Go version for this repo: **1.25.8** (do not bump as part of migration work unless explicitly requested).

---

## Survey inventory in ROSA

Survey usage is centralized in `pkg/interactive/` and `pkg/interactive/confirm/`. Commands call helpers; they rarely import Survey directly.

| Survey primitive | ROSA wrapper / usage | Approx. usage |
|------------------|----------------------|---------------|
| `survey.Input` | `GetString`, `GetInt`, `GetFloat`, `GetIPNet`, `GetCert` | High (~50+ call sites) |
| `survey.Select` | `GetOption`, `GetOptionMode`, `GetInstallerRoleArn`, `GetOidcConfigID` | High (~40+) |
| `survey.MultiSelect` | `GetMultipleOptions` (subnets, security groups, tuning configs, etc.) | Medium (~10+) |
| `survey.Confirm` | `GetBool`, `confirm.Prompt` / `Confirm` | High (~30+ confirm, ~40+ bool) |
| `survey.Password` | `GetPassword` (login, IDP secrets) | Low (~10) |
| `survey.AskOne` | All of the above (one prompt at a time) | Every interactive prompt |
| `survey.WithValidator` + `Validator` | `pkg/interactive/validation.go` (`RegExp`, `ARNValidator`, `IsCIDR`, etc.) | Widespread |
| `survey.TransformString` | `HandleEscapedEmptyString` on `GetString` | `GetString` only |
| `core.DisableColor` | Wired to `pkg/color.UseColor()` | Every prompt |
| `terminal.NewAnsiStdout` | Survey output on stdout | Every prompt |
| `PrintHelp` + `core.RunTemplate` | Instructional help before IDP / upgrade prompts | ~10 call sites |
| `survey.Required` / `MaxLength` | Required fields, prefix length, etc. | Common |

ROSA does **not** use `survey.Ask` (multi-question forms), `survey.Editor`, or multiline Survey prompts.

---

## Survey → Bubble Tea equivalence table

Legend:

- ✅ **Good parity** — same behavior with reasonable effort; UX may still differ slightly.
- ⚠️ **Different UX or gap** — functionally achievable but users will notice a change, or Bubbles has no drop-in widget.

| # | Survey component | ROSA behavior today | Bubble Tea / Bubbles equivalent | Parity | Notes |
|---|------------------|---------------------|----------------------------------|--------|-------|
| 1 | **`survey.Input`** | Single-line text; optional “(optional)” in label; default prefilled | **`bubbles/textinput`** | ✅ Good | Validation on Enter must be implemented in the model (see `user-role-bubble` wizard). |
| 2 | **`survey.Input`** (numeric) | `GetInt` / `GetFloat` parse string after input | **`textinput`** + parse in app code | ✅ Good | No numeric widget; user types a number as today. |
| 3 | **`survey.Input`** (CIDR / cert path) | `GetIPNet` / `GetCert` = Input + custom validators | **`textinput`** + same validators | ✅ Good | Validation logic lives in `pkg/interactive/validation.go`; reuse on submit. |
| 4 | **`survey.Select`** | Arrow keys, one choice; optional **“Skip”** option injected (`consts.SkipSelectionOption`) | **`bubbles/list`** (single-select) | ⚠️ Different UX | List is a scrollable panel with filter/pagination, not Survey’s compact inline prompt. **Reimplement “Skip”** (extra item → return `""`). |
| 5 | **`survey.Select`** (long lists: versions, regions, instance types) | Inline select; lists can be very long | **`list`** with fuzzy filter + pagination | ⚠️ Different UX (often better) | More capable than Survey; **looks and feels different**. |
| 6 | **`survey.MultiSelect`** | Space to toggle, Enter to confirm; validators use `[]core.OptionAnswer` | **No built-in Bubbles component** | ⚠️ Missing / custom | Build a **custom toggle-list model** or add another library. Highest migration cost for cluster / machinepool flows. |
| 7 | **`survey.Confirm`** | `GetBool`, `confirm.Prompt` — `(Y/n)` inline | **Custom `tea.Model`** | ⚠️ Different UX | No official confirm bubble. See `pkg/interactive/bubbletea/confirm.go`. Logic is easy; appearance differs from Survey. |
| 8 | **`survey.Password`** | Masked input | **`textinput`** with `EchoMode = EchoPassword` | ✅ Good | Supported in Bubbles. |
| 9 | **`survey.AskOne`** | Sequential prompts; each blocks the CLI | **`tea.NewProgram`** per step or one multi-step model | ⚠️ Different UX | Survey is line-oriented (minimal screen use). Bubble Tea repaints a TUI region. Users will notice. |
| 10 | **`survey.WithValidator`** | Composable validators on submit | **Validation in `Update`** on Enter | ✅ Good (different API) | Reuse rules from `validation.go`; wire manually in models. |
| 11 | **`survey.TransformString`** | Normalizes escaped empty strings (`HandleEscapedEmptyString`) | **Post-process `textinput.Value()`** | ✅ Good | Trivial to port. |
| 12 | **`core.DisableColor`** | Honors `--color auto\|never\|always` via `pkg/color.UseColor()` | **`lipgloss`** + explicit color wiring | ⚠️ Different UX | Must connect ROSA `--color` / `NO_COLOR` to lipgloss and Tea output; not a global flag like Survey. |
| 13 | **`terminal.NewAnsiStdout`** | Survey writes prompts to stdout | **`tea.WithOutput(os.Stdout)`** | ✅ Good | Bubble Tea owns the terminal while the program runs. |
| 14 | **`PrintHelp`** | Cyan templated instruction block before prompts | **No direct equivalent** | ⚠️ Missing | Use `lipgloss` / `fmt` before Tea, or a dedicated help step in the model. Used in **IDP** and **upgrade** flows. |
| 15 | **`GetOptionMode`** | Select `auto` / `manual` | **`list`** or custom select model | ✅ Good | Implemented in `user-role-bubble` wizard. |
| 16 | **`GetAddonArgument`** | Dispatches by OCM param type to bool/int/CIDR/option/string | **Compose Bubbles widgets** in a step engine | ⚠️ Different UX | Logic ports; variable-length addon wizards need a generic step runner. |
| 17 | **Non-TTY / CI** | Survey behavior varies; many flows expect a TTY | Bubble Tea **needs a TTY** for full UI | ⚠️ Different UX | Fall back to flag values when non-TTY (see `RunWizard` in POC). Required for all migrated commands. |
| 18 | **`interactive.Enabled()` auto-enable** | Commands auto-enable `-i` when flags missing | Same flag logic; Tea when enabled + TTY | ✅ Good | Not Survey-specific. |
| 19 | **`confirm.Yes()` (`-y`)** | Skips Survey confirm | Skip Tea confirm step | ✅ Good | Implemented in `user-role-bubble`. |
| 20 | **`survey.Editor`** | Not used in ROSA | N/A | — | — |
| 21 | **`survey.Ask` (multi-question)** | Not used in ROSA | Multi-step `tea.Model` | — | Bubble Tea is better suited if multi-field forms are ever needed. |

---

## Why 100% identical UX is not realistic

Functional parity (same questions, defaults, validators, outcomes) is achievable for most ROSA flows. **Visual and interaction parity** — making Bubble Tea feel exactly like Survey — is a separate, much harder goal. Survey and Bubble Tea are built on different terminal models; ROSA’s Survey wrappers encode a specific look and feel that Bubbles does not replicate out of the box.

### Two different terminal models

| Aspect | Survey (ROSA today) | Bubble Tea + Bubbles |
|--------|---------------------|----------------------|
| **Screen use** | One line at a time; prior CLI output stays visible above the prompt | Redraws a TUI region (title, help, input/list, errors); can use alt-screen |
| **Prompt shape** | Compact inline: `? Role prefix: [? for help] (ManagedOpenShift)` | Multi-line block: title, help paragraph, styled input, error below |
| **Library role** | Survey owns prompt rendering via `terminal.NewAnsiStdout` | You own layout in `View()`; widgets are building blocks, not finished prompts |
| **Between prompts** | Each `AskOne` is independent; scrollback accumulates | One multi-step model; **completed answers rendered in `View()`** above the active prompt (see `wizard.go`) |
| **Cancellation** | Ctrl+C / interrupt behavior depends on Survey + signal handling | Esc / Ctrl+C handled in model (`user-role-bubble` treats both as abort) |

Survey feels like **answering questions in a shell transcript**. Bubble Tea feels like **a small terminal UI**, even when only one field is on screen.

### Side-by-side: `create user-role` vs `user-role-bubble`

Same four inputs + confirm in the Survey command; Bubble Tea POC in `pkg/interactive/bubbletea/userrole/`.

**Text input (`GetString`)**

Survey (via `interactive.GetString`):

- Message format: `Question:` with optional `(optional)` suffix
- Default shown inline in the prompt line
- Help on **`?`** during the prompt (Survey built-in)
- Single-line cursor at end of prompt

Bubble Tea (`textinput` + `renderTextStep`):

- Title and help rendered as **separate lines above** the input widget
- Default in the input value / placeholder, not in a Survey-style `(default)` suffix
- **No `?` key** unless implemented manually in `Update`
- Bubbles styling (borders, cursor blink) — not Survey’s `?` prefix line

**Select / mode (`GetOptionMode` → `survey.Select`)**

Survey:

- Inline list: arrow keys move highlight on **one or two lines** of options
- Optional “Skip” injected as first option with explanatory text in the question
- Default called out as `default = 'auto'` in the prompt when set

Bubble Tea (`bubbles/list`):

- **Panel UI**: title bar, pagination, keybinding help footer (`↑/↓`, `enter`, etc.)
- Two-item mode list still shows list chrome — disproportionate for small enums
- Different mental model: “pick from a list widget” vs “answer a select question”

**Confirm (`confirm.Prompt` → `survey.Confirm`)**

Survey:

- Standard `(Y/n)` / `(y/N)` inline confirm on one line
- Integrated with Survey color and terminal helpers

Bubble Tea (`confirm.go`):

- Custom model; POC renders `message (Yes) [y/N]:` — close in wording, still not Survey’s renderer
- Separate `tea.NewProgram` run from the wizard — extra screen clear/redraw vs Survey’s back-to-back `AskOne`

**Color (`--color` / `NO_COLOR`)**

Survey: `core.DisableColor = !color.UseColor()` before every prompt — global, consistent.

Bubble Tea: lipgloss styles must be wired per model; POC **does not wire `--color` yet**. Even after wiring, palette and emphasis will not match Survey byte-for-byte.

### Features with no drop-in Bubble Tea equivalent

These block “100% equal” UX without custom work:

1. **`?` help during input** — Survey’s `Help` field; Bubbles `textinput` has no equivalent. Reimplement key handling and a help overlay or extra line.
2. **`PrintHelp` instruction blocks** — Cyan templated paragraphs before IDP/upgrade prompts (`core.RunTemplate`). Must be re-created with lipgloss or plain `fmt` before/alongside Tea.
3. **`survey.MultiSelect`** — Space-to-toggle, Enter-to-finish; no Bubbles v0.20 widget. Custom model required; cannot match Survey’s look without building it.
4. **`GetOption` “Skip” semantics** — Domain-specific option injection and prompt wording; easy to miss when switching to `list`.
5. **Long inline selects** — Survey scrolls a compact select; `list` + filter is **better for search** but **not the same interaction**.

### What you can reasonably target

| Goal | Realistic? | Notes |
|------|------------|-------|
| Same answers, validation, and command outcomes | **Yes** | Reuse `pkg/interactive/validation.go` and flag defaults |
| Same question text and help *content* | **Yes** | Copy strings from existing `interactive.Input` |
| Same prompt *layout* and key hints | **No** (without heavy customization) | Would mean re-skinning Bubbles or avoiding them |
| Same scrollback / transcript behavior | **Partial** | Avoid alt-screen; accept redraw differences |
| Same accessibility profile | **Unknown** | TUI vs line prompts differ for assistive tech |
| Identical appearance with `--color never` | **Hard** | Both disable color; spacing and widgets still differ |

### Implications for migration planning

- Treat **functional parity** as the default acceptance bar for migrated commands.
- Treat **UX parity** as explicit scope: either accept intentional UI improvement (filterable lists) or budget engineering to mimic Survey (custom views, shared prompt templates, `--color` wiring, `?` help, PrintHelp replacement).
- **Do not promise** users “nothing will change” unless the PR includes a UX review against the Survey command and documents remaining differences.
- The POC (`user-role-bubble`) is intentionally **Bubble Tea-native**, not Survey-cloned — use it to prove testability and widget composition, not pixel-perfect parity.

For testing implications of the two approaches, see [survey-bubbletea-migration-testing.md](./survey-bubbletea-migration-testing.md).

---

## Infrastructure and cross-cutting concerns

| Concern | Survey today | Bubble Tea approach | Warning |
|---------|--------------|---------------------|---------|
| **Prompt style** | Inline, one question under prior output | TUI redraw; optional alt-screen (`tea.WithAltScreen`) | ⚠️ Different interaction model |
| **Help text (`Help` field)** | Press `?` during prompt | List has a key help bar; textinput needs custom `?` handling | ⚠️ No built-in `?` on textinput |
| **Optional fields** | “(optional)” in question text | Same label pattern or allow empty Enter | ✅ |
| **Default values** | Shown inline in prompt | `textinput.SetValue()` / pre-selected list item | ✅ |
| **Long-running work + spinner** | `briandowns/spinner` elsewhere | **`bubbles/spinner`** inside Tea model | ✅ Can unify later |
| **Accessibility** | Simple line prompts | Full TUI; screen reader behavior differs | ⚠️ Evaluate for spike success criteria |
| **Testing** | Survey UI hard to unit test | Models testable with `tea.NewProgram(..., tea.WithInput(...))` | ✅ Better test story once models exist |

---

## What `rosa create user-role-bubble` proves

Reference POC: dry-run mirror of `rosa create user-role` using Bubble Tea only (no Survey in this command path).

| Survey piece in `create user-role` | `user-role-bubble` implementation |
|-----------------------------------|-----------------------------------|
| `GetString` ×3 (prefix, boundary, path) | `bubbles/textinput` wizard steps; prior answers kept in `completed` slice and shown in `View()` |
| `GetOptionMode` | `bubbles/list` |
| `confirm.Prompt` | `pkg/interactive/bubbletea/confirm.go` |
| Validators | Reused `aws.*` / regex checks on Enter |
| `core.DisableColor` | **Not wired yet** | ⚠️ TODO for full parity |
| `PrintHelp` | Not used in this command | — |
| AWS / OCM writes (auto mode) | **Faked** — prints success + link command | POC scope only |

---

## Highest-risk gaps for a full Survey replacement

1. ⚠️ **`survey.MultiSelect`** — subnets, security groups, tuning/kubelet configs, log-forwarding pod groups. No Bubbles v0.20 component; requires custom model work.
2. ⚠️ **`PrintHelp`** — IDP and upgrade flows show multi-step instructions; must be reimplemented in or before Tea.
3. ⚠️ **Overall UX shift** — Survey is minimal and inline; Bubble Tea is a TUI. Even with functional parity, **feel will differ**. See [Why 100% identical UX is not realistic](#why-100-identical-ux-is-not-realistic) above.
4. ⚠️ **`GetOption` “Skip” option** — domain-specific (`consts.SkipSelectionOption`); easy to miss during migration.
5. ⚠️ **Color flag integration** — connect `rosa --color` to lipgloss / Tea output explicitly.
6. ⚠️ **Very long selects** (OpenShift versions, regions) — list + filter improves power but changes UX vs Survey’s inline select.

---

## Suggested migration order

Aligned with interactive complexity in `docs/cli-paths.md` (Table 2).

| Priority | Prompt types | Bubble Tea effort |
|----------|--------------|-------------------|
| 1 (POC done) | Input + Select + Confirm | **Low** — `user-role-bubble` |
| 2 | Mode-only + single ARN/string commands | **Low** |
| 3 | Password, cert path, CIDR | **Low** (textinput variants) |
| 4 | Bool / confirm across delete and upgrade | **Low–medium** (shared confirm model) |
| 5 | IDP flows | **Medium** (+ `PrintHelp`) |
| 6 | Addon dynamic parameters (`GetAddonArgument`) | **Medium** |
| 7 | Cluster / machinepool | **High** (multi-select + branching + many steps) |

---

## Agent and contributor guidelines

When migrating a command from Survey to Bubble Tea:

1. **Keep Cobra in `cmd/`** — put Tea models and step logic in `pkg/interactive/bubbletea/` (or a focused subpackage).
2. **Show prior answers in multi-step wizards** — append each accepted answer to a `completed` slice in the model and render it at the top of `View()` so long flows stay reviewable (Survey-like transcript behavior).
3. **Reuse validators** from `pkg/interactive/validation.go` where possible; do not reimplement ARN/CIDR/cert rules.
4. **Preserve flag contracts** — update `cmd/rosa/structure_test/command_args/**` when flags change.
5. **Preserve non-interactive paths** — flags-only usage must work without a TTY; do not require Tea for CI/scripting.
6. **Do not weaken tests** — add model tests where behavior is non-trivial.
7. **Wire `--color`** before claiming parity with existing Survey UX.
8. **Document UX differences** in PR notes when using list/filter UI for long selects or custom multi-select models.

When evaluating a prompt type for Bubble Tea:

- Check this table for ⚠️ rows first.
- Inspect call sites in `pkg/interactive/interactive.go` and the command’s `cmd/` + `pkg/` helpers.
- Compare with the `user-role-bubble` wizard and confirm helpers as the canonical small example.

---

## Maintenance

Update this document when:

- New Survey wrappers or prompt types are added to `pkg/interactive/`.
- Bubble Tea / Bubbles versions change in `go.mod`.
- A command is migrated and establishes a new reusable pattern (e.g. multi-select model, `PrintHelp` replacement).
- POC findings from `ROSAENG-4069` change the go/no-go recommendation.
