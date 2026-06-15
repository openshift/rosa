# Migrating a Survey interactive flow to Bubble Tea

Step-by-step technical guide based on the **`rosa create machinepool-bubble`** spike. Use this when migrating the next Survey-driven command from `pkg/interactive` to Bubble Tea.

**Related:** [golden-path.md](./golden-path.md) (production flow, golden-path script, native vs workaround tables), [manual-testing.md](./manual-testing.md).

---

## Scope note (this spike)

This demo was asked to use **fake/dummy data** — no real OCM/AWS calls, fixture lists instead of live API fetches, and a dry-run success message. The **Bubble Tea mechanics below apply to a real command migration**; only the data layer (`pkg/machinepooldemo/fixtures.go`, `PrintSuccess`) is demo-specific.

Pairing **`machinepool-demo`** (Survey) and **`machinepool-bubble`** (Bubble Tea) on the same golden path lets you compare behavior side by side before touching production `rosa create machinepool`.

---

## Libraries

| Library | Role in ROSA migration |
|---------|-------------------------|
| [`bubbletea`](https://github.com/charmbracelet/bubbletea) | `tea.Model` program loop: `Init`, `Update`, `View`, `tea.Program.Run`, `tea.Println`, `tea.Quit` |
| [`bubbles/textinput`](https://github.com/charmbracelet/bubbles) | `GetString`, `GetInt` (parse after Enter) |
| [`bubbles/list`](https://github.com/charmbracelet/bubbles) | `GetOption`, `GetBool` (as Yes/No list) |
| [`lipgloss`](https://github.com/charmbracelet/lipgloss) | ANSI styling (Survey colors, list chrome) |
| [`teatest`](https://github.com/charmbracelet/x/exp/teatest) | Program-level tests (`user-role-bubble` reference) — optional per command |

There is **no** `bubbles` multi-select. `GetMultipleOptions` needs a custom model or shared helper (see [multiselect.go](../../pkg/interactive/bubbletea/machinepoolbubble/multiselect.go)).

---

## Step-by-step: next command migration

### 1. Inventory the Survey flow in production code

1. Find the interactive entrypoint (for machine pool: `CreateNodePools` in `pkg/machinepool/machinepool.go`).
2. List every `interactive.Get*` call in **execution order** (branching included).
3. Classify each step:

   | Survey helper | Bubble Tea direction |
   |---------------|----------------------|
   | `GetString` | `bubbles/textinput` |
   | `GetInt` | `textinput` + `strconv.Atoi` + same validators |
   | `GetOption` | `bubbles/list` + `newROSAOptionList` (Skip/default text) |
   | `GetBool` | `bubbles/list` with Yes/No items (or confirm model for one-off) |
   | `GetMultipleOptions` | Custom multi-select model |
   | `confirm.Prompt` | `pkg/interactive/bubbletea` confirm (see `user-role-bubble`) |

4. Document a fixed **golden path** script ([golden-path.md](./golden-path.md)) for manual comparison.

**Heads-up:** If production order changes in `machinepool.go`, update the wizard `step` enum, golden path doc, and manual-testing script together.

---

### 2. Keep Cobra thin; put the wizard in `pkg/`

| Layer | Responsibility |
|-------|----------------|
| `cmd/create/<command>bubble/cmd.go` | Cobra wiring, `interactive.AddFlag`, call `RunWizard`, reporter errors, `os.Exit` |
| `pkg/interactive/bubbletea/<command>/` | `tea.Model`, prompts, validation, result struct |
| `pkg/<command>/` (optional) | Shared validators, result types, non-interactive helpers |

**Why:** Matches ROSA rules (`cmd/` = Cobra, `pkg/` = logic). The production command can later call the same wizard package without duplicating prompts.

**Heads-up:** Register the command in `cmd/create/cmd.go` and `cmd/rosa/structure_test/command_structure.yml` + `command_args/.../command_args.yml` when flags change.

---

### 3. Define a result struct and step enum

```text
wizardModel
  step      wizardStep   // explicit state — Survey had implicit order in nested calls
  result    Result       // accumulated answers
  text      textinput.Model
  list      list.Model
  multi     multiSelectModel   // only if GetMultipleOptions exists
  errMsg    string
  done / aborted bool
```

- One `wizardStep` constant per Survey prompt (machine pool: 24 steps).
- `initStep(step)` configures the active widget (new text input, new list, or new multi-select).
- **Why enum + `initStep`:** Survey encodes order in call stack; Bubble Tea needs explicit state for branching (subnet vs AZ, autoscaling, optional skips).

**Heads-up:** Use **pointer receiver** on `Update`/`Init`/`View` and run `tea.NewProgram(&model, ...)`. Returning `*wizardModel` from sub-handlers avoids `unexpected wizard result` at `FinalModel()`.

---

### 4. Implement `RunWizard`

```text
RunWizard() (Result, error)
  1. Require TTY (optional for production; demo enforces it)
  2. model := newWizardModel()
  3. final, err := tea.NewProgram(&model, tea.WithOutput(os.Stdout)).Run()
  4. Assert final.(*wizardModel) — see wizardResult() in transcript.go
  5. Return result or cancellation error
```

**Why `tea.WithOutput(os.Stdout)`:** Same stdout as Survey; required for `tea.Println` transcript lines.

**Heads-up:** Do **not** enable `tea.WithAltScreen()` unless you intend to — `tea.Println` history is suppressed on alt screen.

---

### 5. Map each Survey primitive to Update handlers

Split `Update` by active widget:

```text
Update(msg)
  → updateMulti(msg)   // security groups, tuning, kubelet
  → updateList(msg)    // GetOption, GetBool
  → updateText(msg)    // GetString, GetInt
```

On **Enter**:

1. Read value from `text`, `list.SelectedItem()`, or multi-select completion.
2. Run **production validators** (same functions Survey uses — e.g. `machinepool.MachinePoolKeyRE`, `mpHelpers.LabelValidator`).
3. On error: set `errMsg`, stay on step (Survey re-prompt behavior).
4. On success: write `result`, `recordAnswer` (transcript), `initStep(next)`.

**Why separate handlers:** Each bubbles component has different `Update` signatures and Enter semantics; keeps list filter logic out of text steps.

**Heads-up:** If you validate in only one path (e.g. list but not result struct), golden-path and production paths diverge.

---

### 6. Wire `GetOption` semantics (Skip, default, title)

Survey logic lives in `interactive.GetOption` (`pkg/interactive/interactive.go`). The demo mirrors it in:

- `option_prompt.go` — `buildROSAOptionPrompt`, `rosaOptionInput`
- `list_helpers.go` — `newROSAOptionList`

**Why a helper instead of inlining:** Same rules as production (prepend `Skip`, optional message, `default = '…'`, title suffix `:`). One place to stay aligned with `GetOption`.

On Enter, treat `Skip` like Survey: empty string in `result`, transcript shows `Skip` (`isSkipSelection`).

**Heads-up:** If `interactive.GetOption` changes optional wording or Skip behavior, update `buildROSAOptionPrompt` and [golden-path.md](./golden-path.md) native vs workaround table.

---

### 7. Single-select list: native `bubbles/list` + ROSA styling

| File | Purpose |
|------|---------|
| `list_helpers.go` | `newSelectList`, height/filter/pagination, `maybeStartTypeToFilter` |
| `survey_delegate.go` | `surveySelectDelegate` — cyan `>` focus (implements `list.ItemDelegate`) |
| `option_prompt.go` | Question/options/default assembly |

**Choices:**

| Decision | Chose | Over |
|----------|-------|------|
| Single select widget | `bubbles/list` | Hand-rolled select — keeps filter, pagination, **teatest** ↑/↓/Enter |
| Focus indicator | Custom `ItemDelegate` | Only `Styles` tweak — default delegate uses pink `│` border, not Survey `>` |
| Title chrome | Bold title, no purple `Styles.Title` background | Inline `?` + prompt in custom `View()` — layout workaround, harder to test |
| Filter entry | `/` native + `maybeStartTypeToFilter` | Force users to only use `/` — Survey allows type-to-filter immediately |

**Heads-up:** Replacing `bubbles/list` with a custom select → update teatest key sequences and golden-path **native vs workaround** classification.

---

### 8. Multi-select: custom model (no native widget)

`multiselect.go` — `multiSelectModel` with ↑/↓, **space** toggle, Enter confirm.

**Why custom:** Bubbles has no `GetMultipleOptions` equivalent. Shared pattern for all three machine-pool multi-selects.

**Heads-up:**

| If you change… | Also update… |
|----------------|--------------|
| Keys (e.g. space → `x`) | `manual-testing.md`, teatest scripts, workaround table |
| Max selections (kubelet = 1) | Validator `machinepool.ValidateKubeletConfig` on Enter |
| Reuse across commands | Consider moving to `pkg/interactive/bubbletea/multiselect` — teatest imports path |

Classification: **workaround** (not natively supported by Bubble Tea). Still teatest-able with different keys than list.

---

### 9. Prior answers / scrollback

Survey appends each Q&A to terminal scrollback. Bubble Tea redraws one panel.

**Approach:** `recordAnswer` → `tea.Println(surveyTranscriptLine(...))` in `transcript.go`.

**Why `tea.Println`:** Documented API for persistent output above the program; survives re-renders. Lipgloss + `pkg/color.UseColor()` matches Survey icon colors on completed lines.

**Why not only `View()` summary:** Redraw wipes in-panel history; empty `View()` on `done` clears the screen on exit.

**Heads-up:** Light workaround — not identical to Survey layout. Putting full history only in `View()` without `tea.Println` changes what users see and what teatest `WaitFor` matches.

---

### 10. Reuse production validators; fake only I/O

| Reuse from production | Demo fake |
|----------------------|-----------|
| `machinepool.MachinePoolKeyRE`, `ReplicaSizeValidation`, label/taint/tag validators, `ocm.ValidateHttpTokensValue`, etc. | `machinepooldemo/fixtures.go` option lists |
| `interactive.NodePoolRootDiskSizeValidator()` | No `CreateNodePool` API call |
| Prompt order from `CreateNodePools` | `machinepooldemo.PrintSuccess` dry-run output |

**Why:** Proves Bubble Tea can enforce the same rules; fixtures only remove cloud dependencies.

**Heads-up:** Production migration must swap fixtures for real `Runtime` / OCM / AWS calls **after** the wizard returns `Result` — wizard package should not import AWS clients unless needed for live option lists (e.g. instance types).

---

### 11. Register CLI and verify

1. `cmd/create/cmd.go` — `Cmd.AddCommand(...)`
2. `cmd/rosa/structure_test/command_structure.yml`
3. `cmd/rosa/structure_test/command_args/rosa/create/<command>/command_args.yml`
4. `make rosa` — build
5. Manual run: [manual-testing.md](./manual-testing.md)
6. Optional: `teatest` on a short path (`pkg/interactive/bubbletea/userrole/wizard_teatest_test.go` as template)

**Heads-up:** `teatest` `FinalModel` must match pointer type. `WaitFor` strings must match list titles (full `GetOption` prompt text after Skip work).

---

## Files added/changed (machine pool bubble spike)

### New — Bubble Tea wizard (`pkg/interactive/bubbletea/machinepoolbubble/`)

| File | Role |
|------|------|
| `wizard.go` | `tea.Model`: step enum, `Update`/`View`, branching, validation, `RunWizard` |
| `list_helpers.go` | List construction, filter shim, pagination height |
| `option_prompt.go` | `buildROSAOptionPrompt` — mirrors `interactive.GetOption` |
| `survey_delegate.go` | `list.ItemDelegate` — Survey-like `>` focus |
| `transcript.go` | `tea.Println` transcript, `wizardResult`, `mergeCmds` |
| `multiselect.go` | Custom `GetMultipleOptions` UI |

### New — Demo shared (`pkg/machinepooldemo/`)

| File | Role |
|------|------|
| `fixtures.go` | Fake versions, subnets, SGs, instance types |
| `survey.go` | Survey golden-path runner (reference) |
| `result.go` | Collected answers struct |
| `output.go` | Dry-run success message |
| `validate.go` | Taint validator wrapper |
| `goldenpath.go` | Demo-only re-prompt messages |

### New — Commands (`cmd/create/`)

| File | Role |
|------|------|
| `machinepoolbubble/cmd.go` | `rosa create machinepool-bubble` |
| `machinepooldemo/cmd.go` | `rosa create machinepool-demo` (Survey baseline) |

### Changed

| File | Change |
|------|--------|
| `cmd/create/cmd.go` | Register both demo commands |
| `cmd/rosa/structure_test/command_structure.yml` | Command tree |
| `cmd/rosa/structure_test/command_args/rosa/create/machinepool-bubble/command_args.yml` | `-i` flag |
| `cmd/rosa/structure_test/command_args/rosa/create/machinepool-demo/command_args.yml` | `-i` flag |

### Simpler reference migration

`rosa create user-role-bubble` — four text/list steps, confirm, no multi-select: `pkg/interactive/bubbletea/userrole/`, teatest in `wizard_teatest_test.go`. Start there for a first migration; machine pool adds list helpers, multi-select, and 24-step state.

---

## Heads-up summary (decisions → consequences)

| Decision | Teatest | Native Bubble Tea? | Update also |
|----------|---------|-------------------|-------------|
| Use `bubbles/list` for select | ↑/↓, Enter, `/`, runes | Yes | — |
| Use custom `ItemDelegate` for `>` color | Output text changes | Yes (delegate API) | `survey_delegate.go`, manual-testing |
| Use `buildROSAOptionPrompt` / Skip | Extra ↓ when Skip is first row | Yes (app logic) | `option_prompt.go`, `interactive.GetOption` |
| Use `maybeStartTypeToFilter` | Send `KeyRunes` | Light workaround | `list_helpers.go`, workaround table |
| Use `tea.Println` transcript | `WaitFor` on `? …` lines | Yes (`tea.Println`) | `transcript.go` |
| Custom `multiSelectModel` | **space** + Enter, not list keys | **No** — workaround | `multiselect.go`, workaround table |
| Hand-rolled single-select | Custom key map | **No** — avoid | teatest, workaround table |
| Pointer vs value `wizardModel` | `*wizardModel` in `FinalModel` | Yes | `RunWizard`, tests |
| Alt screen enabled | `tea.Println` invisible | Yes | Program options |
| Demo fixtures vs live API | N/A | N/A | Production command I/O after wizard |
| Change prompt order in production | Step enum + all docs | N/A | `wizard.go`, golden-path, manual-testing |

---

## Keep documentation in sync

Whenever `machinepool-bubble` implementation changes, update the **Bubble Tea demo: native vs workaround** table in [golden-path.md](./golden-path.md). That table is the short classification reference; this document is the how-to for the next migration.

---

*Based on ROSAENG-4069 spike, branch `ROSAENG-4069-bubble-tea`.*
