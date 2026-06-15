# Survey → Bubble Tea migration (ROSAENG-4069)

Documentation and recordings for the ROSA CLI spike that evaluates replacing [Survey](https://github.com/AlecAivazis/survey/v2) with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Bubbles](https://github.com/charmbracelet/bubbles).

**Branch:** `ROSAENG-4069-bubble-tea`  
**Jira:** ROSAENG-4069

Build the CLI from the repo root before trying demo commands:

```bash
make rosa
./rosa create <demo-command> -i
```

Use `./rosa` from the build tree, not an older binary on `$PATH`.

---

## POC videos

Screen recordings of the HCP **create machinepool** golden path (24 interactive steps, fake fixtures, production validators, no real OCM/AWS create). Both videos walk the same answers; they show two **migration UX strategies**, not two levels of project risk.

| Strategy | File | Command | Intent |
|----------|------|---------|--------|
| **Survey parity** | [poc/videos/poc-survey-parity.mp4](./poc/videos/poc-survey-parity.mp4) | `rosa create machinepool-bubble -i` | De-risk migration: stay as close to Survey as practical (green `?`, transcript-style completed answers, Survey-like list focus, `(optional)` labels, custom multi-select where Bubbles has no widget). |
| **UX-first** | [poc/videos/poc-ux-first.mp4](./poc/videos/poc-ux-first.mp4) | `rosa create machinepool-bubble-new -i` | Target experience: Bubble Tea used fully (Charm lipgloss palette, step progress header, in-view answer summary, native `bubbles/list` checkbox delegate for multi-select with filter and pagination, contextual help on text steps). |

**Survey baseline** (not recorded here, same golden path): `rosa create machinepool-demo -i` — production Survey wrappers and styling for side-by-side comparison.

For product discussions: lead with **Survey parity** to show migration is safe; use **UX-first** to show optional upside after the library move.

**Recording:** Terminal sessions in [Terminator](https://gnome-terminator.org/); capture with Screenshot UI (Fedora); edited with [OpenShot](https://www.openshot.org/).

---

## Directory layout

```text
survey-bubbletea-migration/
├── README.md                              ← this file
├── survey-bubbletea-migration.md          ← migration reference (Survey inventory, mapping, risks)
├── survey-bubbletea-migration-testing.md  ← testing strategy (model tests vs teatest)
├── survey-bubbletea-migration-pm-brief.md ← PM-facing spike summary
└── poc/
    ├── golden-path.md                     ← production flow + golden-path script + native vs workaround tables
    ├── survey-to-bubbletea-migration.md   ← step-by-step how-to (machine pool spike)
    └── videos/
        ├── poc-survey-parity.mp4          ← Survey-parity Bubble Tea demo
        └── poc-ux-first.mp4               ← UX-first Bubble Tea demo
```

---

## Document guide

| File | Audience | Contents |
|------|----------|----------|
| [survey-bubbletea-migration.md](./survey-bubbletea-migration.md) | Contributors, agents | Survey usage in ROSA, primitive mapping to Bubbles, migration patterns, dependency notes, command inventory pointers. Start here for **what** to migrate. |
| [survey-bubbletea-migration-testing.md](./survey-bubbletea-migration-testing.md) | Contributors | Why Survey has no flow tests today; model-level `Update` tests vs optional `teatest` smoke; reference: `user-role-bubble`. |
| [survey-bubbletea-migration-pm-brief.md](./survey-bubbletea-migration-pm-brief.md) | Product / leadership | Why Survey is risky, why Bubble Tea was spiked, UX and engineering trade-offs, decision framing (informational, not a go/no-go). |
| [poc/golden-path.md](./poc/golden-path.md) | Reviewers, testers | HCP machine pool interactive flow, 24-step demo script, environment prerequisites, **native vs workaround** tables for `machinepool-bubble` and `machinepool-bubble-new`. |
| [poc/survey-to-bubbletea-migration.md](./poc/survey-to-bubbletea-migration.md) | Implementers | Concrete migration recipe from the machine pool spike: packages, wizard structure, list/multiselect, teatest notes. |

Related repo docs outside this folder: [cli-paths.md](../cli-paths.md) (interactive command complexity ranking).

---

## Demo commands on the branch

| Command | Stack | Role in spike |
|---------|-------|----------------|
| `rosa create user-role-bubble` | Bubble Tea | Smallest reference wizard; first teatest pattern |
| `rosa create machinepool-demo -i` | Survey | Golden-path baseline (fixtures only) |
| `rosa create machinepool-bubble -i` | Bubble Tea | **Survey parity** POC ([video](./poc/videos/poc-survey-parity.mp4)) |
| `rosa create machinepool-bubble-new -i` | Bubble Tea | **UX-first** POC ([video](./poc/videos/poc-ux-first.mp4)) |

Implementation packages:

- `pkg/interactive/bubbletea/userrole/` — user role wizard
- `pkg/interactive/bubbletea/machinepoolbubble/` — Survey-parity machine pool wizard
- `pkg/interactive/bubbletea/machinepoolbubblenew/` — UX-first machine pool wizard
- `pkg/machinepooldemo/` — shared fixtures, validators, golden-path runner (Survey demo)

---

## Suggested reading order

1. **PM brief** — context and trade-offs  
2. **POC videos** — Survey parity first, then UX-first  
3. **golden-path.md** — what the demo proves and what is native vs workaround  
4. **survey-bubbletea-migration.md** + **testing** — migrate the next command  
5. **poc/survey-to-bubbletea-migration.md** — follow the machine pool recipe
