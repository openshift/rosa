# ROSA CLI interactive prompts: Survey → Bubble Tea evaluation

**Jira:** ROSAENG-4069  
**Branch:** `ROSAENG-4069-bubble-tea`  
**Audience:** Product management  
**Status:** Spike complete on branch; **no migration decision yet**

---

## Why the ROSA CLI uses Survey today

- **Survey** is a Go library for **interactive terminal prompts** (text, select, multi-select, confirm).
- ROSA uses it in **interactive mode** (`-i`) or when required flags are missing.
- It handles **presentation and input only**; business logic (OCM, AWS, validation) lives elsewhere in `pkg/interactive/`.
- Usage is **broad**: simple prompts through long wizards (cluster create, machine pools, IDP, upgrades). See `developer-docs/cli-paths.md`.

---

## Why staying on Survey is risky

- Survey is **archived and unmaintained** (April 2024). The author points adopters to alternatives, including Bubble Tea.
- ROSA pins **`v2.2.15`** — no path for fixes, security updates, or terminal compatibility work.
- Staying on Survey is a **near-term default**, not a viable long-term strategy.

---

## Why we evaluated Bubble Tea

**Bubble Tea** + **Bubbles** is the actively maintained stack the Survey author and much of the Go CLI ecosystem recommend as the successor.

**Reasons for the spike:**

- **Maintained ecosystem** with ongoing releases.
- **Better fit for complex wizards** — step state, prior answers, branching as explicit models (not ad-hoc prompt chains).
- **Testability** — ROSA has **no automated interactive flow tests** for Survey today; Bubble Tea wizards can be tested per command.
- **Extensibility** when Bubbles has no drop-in widget (e.g. multi-select).

POC commands mirror production flows (same questions, validators, outcomes) with fake data — no real AWS/OCM calls.

---

## Two UX strategies (what the demos show)

| Strategy | Purpose for migration |
|----------|----------------------|
| **Survey parity** | De-risk the move: same flow and outcomes, UI kept as close to Survey as practical. |
| **UX-first** | Show optional upside after the library move (progress, summaries, richer lists). A target experience to phase in per command if desired. |

Screen recordings and commands: [README.md](./README.md) → **POC videos**.

---

## Main concerns and trade-offs

### User experience

- Survey is **line-oriented**; answers accumulate in scrollback.
- Bubble Tea **redraws** multi-line panels — look and feel differ even when questions match.
- **Survey parity** limits surprise; **UX-first** trades familiarity for a clearer wizard. Neither is pixel-identical to Survey without extra custom UI.
- Some patterns (IDP instruction blocks, multi-select) need **custom development** in both approaches.

### Migration scope

- **Phased work**, not a library swap — dozens of interactive call sites across the CLI.
- Suggested order: simple input / select / confirm first; cluster and machine pool last.

### Engineering gaps (machine pool spike)

| Gap | Notes |
|-----|--------|
| **Multi-select** | No Bubbles widget — custom work required (biggest machine-pool risk; three prompts on golden path) |
| **Long selects** | `bubbles/list` + filter — mostly native; UX differs, often stronger |
| **Instruction blocks before prompts** | Not in POC — must be built for some commands |

### Testing

- Bubble Tea enables per-wizard automated tests; effort is per command. Still a clear improvement vs Survey today.

### Decision summary

| | Stay on Survey | Move to Bubble Tea |
|---|----------------|-------------------|
| Maintenance | None (archived) | Active ecosystem |
| User-visible UX | Unchanged | Manageable via **Survey parity**; **UX-first** optional |
| Engineering effort | Low near-term | High, phased |
| Interactive test coverage | Effectively none | Can be added |
| Complex commands | Dependency rot | Upfront cost; clearer long-term path |

---

## Demo

**Machine pool golden path** (24 steps, HCP, fake fixtures):

| Strategy | Video | Command |
|----------|-------|---------|
| **Survey parity** | [poc-survey-parity.mp4](./poc/videos/poc-survey-parity.mp4) | `rosa create machinepool-bubble -i` |
| **UX-first** | [poc-ux-first.mp4](./poc/videos/poc-ux-first.mp4) | `rosa create machinepool-bubble-new -i` |

Survey baseline: `rosa create machinepool-demo -i`.

**User role** (smaller reference): `rosa create user-role-bubble` — prefix, permissions boundary, path, creation mode.

**Try on branch:** `git checkout ROSAENG-4069-bubble-tea` → `make rosa` → `./rosa create machinepool-bubble -i`

**Further reading:** [golden-path.md](./poc/golden-path.md), [survey-bubbletea-migration.md](./survey-bubbletea-migration.md)

---

*Prepared from ROSAENG-4069 spike work. Informational only — not a go/no-go recommendation.*
