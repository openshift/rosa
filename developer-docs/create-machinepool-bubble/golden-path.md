# HCP `create machinepool` — golden path for the Bubble Tea demo

Reference scenario for **`ROSAENG-4069`** on branch `ROSAENG-4069-bubble-tea`.

**Related docs**

- [interactive-flow.md](./interactive-flow.md) — step-by-step Survey behavior
- [interactive-flow-diagram.md](./interactive-flow-diagram.md) — Mermaid flowchart
- [manual-testing.md](./manual-testing.md) — inputs, validations, and test sessions for the demo commands
- [survey-to-bubbletea-migration.md](./survey-to-bubbletea-migration.md) — how to migrate the next Survey command to Bubble Tea

---

## What is the golden path?

The **golden path** is the longest **happy-path** walk through `rosa create machinepool -i` on an **HCP cluster** that:

- Hits **every Survey primitive** used in `CreateNodePools`
- Takes **every answer-driven branch that adds prompts** (not exits)
- Includes **all three** `GetMultipleOptions` steps when the cluster and AWS environment support them
- Ends in a successful **`CreateNodePool`** (or a faked success in the Bubble Tea POC, mirroring `user-role-bubble`)

It is the **stress-test scenario** for the machine-pool Bubble Tea demo—not the path most customers run daily.

---

## Why we use this path

| Reason | Detail |
|--------|--------|
| **Representative complexity** | Machine pool create is among the hardest interactive flows in ROSA (see `developer-docs/cli-paths.md`). The demo must prove Bubble Tea on a real command, not a toy wizard. |
| **MultiSelect coverage** | Survey `GetMultipleOptions` has **no Bubbles equivalent**. Security groups, tuning configs, and kubelet config are the main migration risks; the golden path forces all three. |
| **Branching** | Subnet/AZ tree and autoscaling yes/no show that a single linear wizard is not enough—models need explicit step state. |
| **Input variety** | Exercises `GetString`, `GetOption`, `GetBool`, `GetInt`, and `GetMultipleOptions` in one run—closest to “full command coverage” in one session. |
| **POC comparison** | After `user-role-bubble` (input + select + confirm), machine pool is the next credible step toward cluster-adjacent flows without migrating all of `rosa create cluster`. |
| **Test design** | One documented scenario gives reviewers a fixed script for manual runs, GIFs, and automated model/teatest tests. |

We **do not** use this path because every user follows it—it is the **maximum** interactive surface for evaluation.

---

## Assumptions (aligned with the flowchart)

- **100% interactive** — `-i` / `--interactive`; no flags pre-filled
- **HCP only** — `CreateNodePools`; classic machine pools out of scope
- **Commercial AWS** — not FedRAMP/GovCloud (see below)
- **Validation retries** — invalid answers re-prompt on the same step (not part of the golden path script)

---

## Answer-driven choices (golden path)

| Decision | Choose | Effect |
|----------|--------|--------|
| FedRAMP / GovCloud? | **No** | Image type, capacity reservation, and kubelet prompts run |
| Image type | **Default** *(or Windows if LI preview is active)* | Same prompt count; Windows adds a warn only |
| Select subnet for hosted pool? | **No** | Routes through availability zone |
| Subnets in chosen AZ? | **Multiple** | Extra subnet `GetOption` after AZ |
| Enable autoscaling? | **Yes** | Min + max replicas (not single replica count) |
| Day-2 security groups supported? | **Yes** (cluster version) | Security group multi-select can run |
| SGs available in VPC? | **Yes** | **`GetMultipleOptions`: security groups** |
| Tuning configs on cluster? | **Yes** | **`GetMultipleOptions`: tuning configs** |
| Capacity Reservation ID | **Empty** | Full preference list (`none` / `only` / `open`) |
| Kubelet configs on cluster? | **Yes** | **`GetMultipleOptions`: kubelet config** (max one) |
| Cluster autoscaler `MaxNodesTotal` | **Set** *(optional)* | Info messages before create; not an extra prompt |

---

## Prompt sequence (24 Survey steps)

| # | Primitive | Prompt |
|---|-----------|--------|
| 1 | `GetString` | Machine pool name |
| 2 | `GetOption` | Image type |
| 3 | `GetOption` | OpenShift version |
| 4 | `GetBool` | Select subnet for hosted machine pool? → **No** |
| 5 | `GetOption` | AWS availability zone |
| 6 | `GetOption` | Subnet ID |
| 7 | `GetBool` | Enable autoscaling? → **Yes** |
| 8 | `GetInt` | Min replicas |
| 9 | `GetInt` | Max replicas |
| 10 | `GetString` | Labels |
| 11 | `GetString` | Taints |
| 12 | **`GetMultipleOptions`** | **Additional Machine Pool Security Group IDs** |
| 13 | `GetString` | Tags |
| 14 | `GetOption` | Instance type |
| 15 | `GetBool` | Autorepair |
| 16 | **`GetMultipleOptions`** | **Tuning configs** |
| 17 | `GetString` | Capacity Reservation ID |
| 18 | `GetOption` | Capacity Reservation Preference |
| 19 | **`GetMultipleOptions`** | **Kubelet config** |
| 20 | `GetOption` | IMDSv2 optional / required |
| 21 | `GetString` | Root disk size (GiB or TiB) |
| 22 | `GetString` | Node drain grace period |
| 23 | `GetString` | Max surge |
| 24 | `GetString` | Max unavailable |

Then: optional autoscaler info → OCM `CreateNodePool` → success output.

### Primitive totals

| Primitive | Count |
|-----------|-------|
| `GetString` | 9 |
| `GetOption` | 7 |
| `GetBool` | 3 |
| `GetInt` | 2 |
| **`GetMultipleOptions`** | **3** |

---

## Cluster and environment prerequisites

For the **full** golden path (all 24 prompts), the target cluster/environment needs:

| Requirement | Purpose |
|-------------|---------|
| HCP cluster in **ready** state | Phase 0 pre-checks |
| **Commercial** region (not GovCloud) | Image type, capacity, kubelet blocks |
| OpenShift version supporting **additional day-2 security groups** on node pools | Step 13 multi-select |
| **Non-empty** list of attachable security groups in the VPC | Step 13 multi-select |
| **Tuning configs** already on the cluster (OCM) | Step 16 multi-select |
| **Kubelet configs** already on the cluster (OCM) | Step 19 multi-select |
| VPC private subnets: AZ with **multiple** subnets | Steps 5–6 (longest network branch) |
| Valid instance types for region/AZ/image type | Step 14 |
| AWS + OCM credentials working | Instance type fetch, create API |

If a prerequisite is missing, the diagram **shortens** the path (e.g. no tuning configs → skip step 16). That is valid for a smaller demo but is **not** the full golden path.

---

## Fallback: minimal MultiSelect demo

When the environment cannot satisfy tuning/kubelet/cluster-version gates, use a **reduced** path that still includes **one** `GetMultipleOptions`:

- Same commercial HCP cluster with **day-2 security groups** and **SGs in VPC**
- Accept **skips** for tuning (step 16) and kubelet (step 19) when OCM lists are empty
- Still use **autoscaling yes** and **subnet no → AZ** for reasonable branching

~21 prompts; enough to demo MultiSelect without full cluster prep.

---

## What the Bubble Tea POC mirrors today

Commands: `rosa create machinepool-demo -i` (Survey) and `rosa create machinepool-bubble -i` (Bubble Tea). See [manual-testing.md](./manual-testing.md).

1. **Same question order** as `CreateNodePools` (labels → taints → security groups → tags → instance type in code).
2. **Completed answers in scrollback** — Survey appends `? question: answer` lines; Bubble Tea uses native `tea.Println` with Survey-matched colors (green `?`, bold question, cyan answer).
3. **Fake create** — no real `CreateNodePool`; dry-run success summary only.
4. **Golden-path guardrails** (demo only) — subnet **Yes** and autoscaling **No** re-prompt with a message (production Survey does not loop).

**Survey-style list parity** (optional `Skip`, long prompt title, cyan `>` focus, no purple title bar) is implemented on `machinepool-bubble` via `bubbles/list` styles + delegate — see workarounds table below.

---

## Bubble Tea demo: native vs workaround

**Canonical reference** for `machinepool-bubble` implementation choices. Update this table whenever demo behavior, widgets, or classification changes (not only when `CreateNodePools` changes).

Use this when judging migration risk. **Native** = `tea` / `bubbles` APIs or documented Charm patterns. **Workaround** = extra code to match Survey or fill a Bubbles gap.

**Source files:** `pkg/interactive/bubbletea/machinepoolbubble/` (`wizard.go`, `list_helpers.go`, `option_prompt.go`, `survey_delegate.go`, `transcript.go`, `multiselect.go`).

| Area | Classification | Implementation (`machinepool-bubble`) | Migration / teatest note |
|------|----------------|----------------------------------------|---------------------------|
| Single select (`GetOption`, `GetBool` as list) | **Native** | `bubbles/list` + `newROSAOptionList` (Skip/default prompt) + `surveySelectDelegate` (cyan `>`) | `teatest`: ↑/↓, Enter, `/`, type-to-filter; optional lists start on **Skip** unless default is set |
| Text / int (`GetString`, `GetInt`) | **Native** | `bubbles/textinput` | `teatest`: type + Enter |
| Multi-select (`GetMultipleOptions` ×3) | **Workaround** (no Bubbles widget) | Custom `multiSelectModel` in `multiselect.go` | **Main migration risk** for machine pool; `teatest` still works but uses **space** + Enter, not list keys |
| Prior answers visible | **Light workaround** | `tea.Println` transcript (not Survey scrollback in `View()`) | Native Bubble Tea API; layout differs from Survey redraw model |
| Type-to-filter without `/` | **Light workaround** | `maybeStartTypeToFilter` synthesizes `/` + first key before `list.Update` | `teatest`: send `KeyRunes`; list filter is still native underneath |
| Survey colors on transcript | **Native** | lipgloss + `pkg/color` in `surveyTranscriptLine` | Cosmetic; respects `--color` |
| Golden-path re-prompts | **Demo-only** (not production) | Loop on subnet / autoscaling in demo runners | Not part of production migration |
| Spinner / AWS instance-type fetch | **Out of demo scope** | Fake fixture lists | Production migration must keep real fetch + optional spinner |

### What we are **not** doing (by design)

- **Hand-rolled single-select** replacing `bubbles/list` — would hurt teatest and lose list filter/pagination for long option sets.
- **Pixel-perfect Survey layout** (inline `?` + long prompt + `>` in one scrollback block) — Bubble Tea uses a live panel; functional parity, not identical chrome.
- **Automated teatest for machine pool demo** — optional later; `user-role-bubble` already proves the pattern.

---

## What we are not proving with this path

- Identical Survey UX (layout and TUI differ by design)
- Classic machine pools (`CreateMachinePool`)
- FedRAMP/GovCloud interactive behavior
- Non-interactive flag-only creates
- Edit or delete machine pool flows

---

## Maintenance

- **`Bubble Tea demo: native vs workaround`** — keep in sync with every `machinepool-bubble` change (new prompt types, delegates, workarounds, teatest notes). This table is the migration-risk summary for PM/engineering.
- **Golden path script** — update when `pkg/machinepool/machinepool.go` (`CreateNodePools`), the flowchart, or demo scope changes. Re-validate prerequisites if minimum OpenShift version or feature gates move.
- **Related docs** — [survey-to-bubbletea-migration.md](./survey-to-bubbletea-migration.md) for migration how-to; [manual-testing.md](./manual-testing.md) for runbooks; [survey-bubbletea-migration-pm-brief.md](../survey-bubbletea-migration-pm-brief.md) for PM-level takeaway.
